// chat_server.cpp
// Build dependencies: Crow (crow_all.h), mysql-connector-c++ (cppconn), nlohmann/json
// Example compile (adjust include/lib paths):
// g++ chat_server.cpp -std=c++17 -I/path/to/crow -I/path/to/json -I/usr/include/cppconn -lmysqlcppconn -lpthread -o chat_server

#include <crow/crow_all.h>                // single-header Crow (https://github.com/CrowCpp/Crow)
#include <cppconn/driver.h>
#include <cppconn/connection.h>
#include <cppconn/prepared_statement.h>
#include <cppconn/resultset.h>
#include <cppconn/exception.h>
#include <mysql_driver.h>
#include <nlohmann/json.hpp>

#include <chrono>
#include <ctime>
#include <iostream>
#include <map>
#include <mutex>
#include <optional>
#include <sstream>
#include <string>
#include <vector>

using json = nlohmann::json;
using namespace std::chrono_literals;

struct User {
    long id;
    std::string userName;
    std::chrono::system_clock::time_point lastTimestamp;
};

struct Channel {
    long channel_id;
    std::string channel_type;
    std::string channel_name;
};

struct Membership {
    long membership_id;
    long channel_id;
    long user_id;
};

struct Message {
    long message_id;
    long sender_id;
    long channel_id;
    std::string msg;
    std::chrono::system_clock::time_point created_at;
};

// Global DB connection pointer and mutex
std::unique_ptr<sql::Connection> dbConn;
std::mutex dbMutex;

// WebSocket map: userId -> vector of websocket connections (for simplicity one connection per user in this example)
std::map<long, std::vector<crow::websocket::connection*>> wsMap;
std::mutex wsMutex;

// Helper: convert time_point -> MySQL DATETIME string
std::string timePointToSQL(const std::chrono::system_clock::time_point &tp) {
    std::time_t t = std::chrono::system_clock::to_time_t(tp);
    std::tm tm{};
#if defined(_WIN32) || defined(_WIN64)
    localtime_s(&tm, &t);
#else
    localtime_r(&t, &tm);
#endif
    char buf[64];
    std::strftime(buf, sizeof(buf), "%F %T", &tm);
    return std::string(buf);
}

// Helper: parse rows that return created_at as string to time_point (best effort)
std::chrono::system_clock::time_point parseSQLTime(const std::string &s) {
    std::tm tm{};
    std::istringstream ss(s);
    ss >> std::get_time(&tm, "%Y-%m-%d %H:%M:%S");
    auto tp = std::chrono::system_clock::from_time_t(std::mktime(&tm));
    return tp;
}

// Initialize DB connection
void connectToDb(const std::string &user, const std::string &pass, const std::string &host, const std::string &schema) {
    try {
        sql::mysql::MySQL_Driver *driver = sql::mysql::get_mysql_driver_instance();
        // connection URI: tcp://host:port if needed
        std::string connStr = "tcp://" + host + ":3306";
        dbConn.reset(driver->connect(connStr, user, pass));
        dbConn->setSchema(schema);
        std::cout << "Connected to MySQL\n";
    } catch (sql::SQLException &e) {
        std::cerr << "DB connect error: " << e.what() << "\n";
        throw;
    }
}

// Send message to members over websocket
void sendMessageOverWebSocket(const std::vector<Membership> &members, const std::string &msg, const std::string &userName) {
    std::lock_guard<std::mutex> lk(wsMutex);
    std::string payload = userName + ":" + msg;
    for (const auto &m : members) {
        auto it = wsMap.find(m.user_id);
        if (it != wsMap.end()) {
            for (auto *conn : it->second) {
                if (conn) {
                    try {
                        conn->send_text(payload);
                    } catch (const std::exception &e) {
                        std::cerr << "Error sending websocket message: " << e.what() << "\n";
                    }
                }
            }
        }
    }
}

int main() {
    crow::SimpleApp app;

    // configure DB (adjust creds as required)
    try {
        connectToDb("root", "localhost", "127.0.0.1", "slack");
    } catch (...) {
        std::cerr << "Failed to connect to DB; exiting\n";
        return 1;
    }

    // Serve static files at root (like Go's FileServer)
    CROW_ROUTE(app, "/").methods("GET"_method)([]() {
        crow::response res;
        res.code = 200;
        res.body = "<html><body>Chat server root. Use API endpoints.</body></html>";
        res.add_header("Content-Type", "text/html");
        return res;
    });

    // createUser endpoint: /createUser/{userName}
    CROW_ROUTE(app, "/createUser/<string>").methods("POST"_method, "GET"_method)(
        [](const crow::request &req, crow::response &res, std::string userName) {
            try {
                std::lock_guard<std::mutex> lk(dbMutex);

                std::unique_ptr<sql::PreparedStatement> ps(dbConn->prepareStatement(
                    "SELECT id, userName FROM user WHERE userName = ?"));
                ps->setString(1, userName);
                std::unique_ptr<sql::ResultSet> rs(ps->executeQuery());

                long id = 0;
                std::string foundName;
                if (rs->next()) {
                    id = rs->getInt64("id");
                    foundName = rs->getString("userName");
                }

                if (foundName.empty()) {
                    std::unique_ptr<sql::PreparedStatement> ins(dbConn->prepareStatement(
                        "INSERT INTO user (userName,last_timestamp) VALUES (?, ?)"));
                    using clock = std::chrono::system_clock;
                    auto now = clock::now();
                    ins->setString(1, userName);
                    ins->setString(2, timePointToSQL(now));
                    ins->executeUpdate();
                    std::unique_ptr<sql::Statement> s(dbConn->createStatement());
                    std::unique_ptr<sql::ResultSet> rid(s->executeQuery("SELECT LAST_INSERT_ID() AS id"));
                    if (rid->next()) id = rid->getInt64("id");
                }

                json j;
                j["id"] = id;
                j["userName"] = userName;
                // lastTimestamp not returned unless needed

                res.set_header("Content-Type", "application/json");
                res.write(j.dump());
                res.end();
            } catch (sql::SQLException &e) {
                std::cerr << "createUser DB error: " << e.what() << "\n";
                res.code = 500;
                res.write("fetch try again");
                res.end();
            }
        });

    // createChannel endpoint: /channel/{senderName}/{receiverName}
    CROW_ROUTE(app, "/channel/<string>/<string>").methods("GET"_method, "POST"_method)(
        [](const crow::request &req, crow::response &res, std::string senderName, std::string receiverName) {
            try {
                std::lock_guard<std::mutex> lk(dbMutex);

                auto findUserId = [&](const std::string &name) -> long {
                    std::unique_ptr<sql::PreparedStatement> ps(dbConn->prepareStatement(
                        "SELECT id FROM user WHERE userName = ?"));
                    ps->setString(1, name);
                    std::unique_ptr<sql::ResultSet> rs(ps->executeQuery());
                    if (rs->next()) return rs->getInt64("id");
                    return 0;
                };

                long senderId = findUserId(senderName);
                long receiverId = findUserId(receiverName);

                // Look up common channel between them
                std::unique_ptr<sql::PreparedStatement> q(dbConn->prepareStatement(
                    "SELECT c.channel_id, c.channel_type, c.channel_name "
                    "FROM channel c "
                    "JOIN membership m1 ON c.channel_id = m1.channel_id "
                    "JOIN membership m2 ON c.channel_id = m2.channel_id "
                    "WHERE m1.user_id = ? AND m2.user_id = ?"));
                q->setInt64(1, senderId);
                q->setInt64(2, receiverId);
                std::unique_ptr<sql::ResultSet> rs(q->executeQuery());

                std::vector<json> output;
                long channelId = 0;
                if (rs->next()) {
                    channelId = rs->getInt64("channel_id");
                }

                if (channelId == 0) {
                    // create channel
                    std::unique_ptr<sql::PreparedStatement> ins_ch(dbConn->prepareStatement(
                        "INSERT INTO channel (channel_type, channel_name) VALUES (?, ?)"));
                    ins_ch->setString(1, "DM");
                    ins_ch->setString(2, senderName + "_" + receiverName);
                    ins_ch->executeUpdate();

                    std::unique_ptr<sql::Statement> s(dbConn->createStatement());
                    std::unique_ptr<sql::ResultSet> rid(s->executeQuery("SELECT LAST_INSERT_ID() AS id"));
                    if (rid->next()) channelId = rid->getInt64("id");

                    // create "channel created" message as in Go code
                    std::string channelCreateMsg = "channel created b/w you and " + receiverName;
                    std::unique_ptr<sql::PreparedStatement> ins_msg(dbConn->prepareStatement(
                        "INSERT INTO message (sender_id, channel_id, msg, created_at) VALUES (?, ?, ?, ?)"));
                    using clock = std::chrono::system_clock;
                    auto now = clock::now();
                    ins_msg->setInt64(1, senderId);
                    ins_msg->setInt64(2, channelId);
                    ins_msg->setString(3, channelCreateMsg);
                    ins_msg->setString(4, timePointToSQL(now));
                    ins_msg->executeUpdate();

                    // create membership entries
                    std::unique_ptr<sql::PreparedStatement> ins_mem(dbConn->prepareStatement(
                        "INSERT INTO membership (channel_id, user_id) VALUES (?, ?)"));
                    ins_mem->setInt64(1, channelId);
                    ins_mem->setInt64(2, senderId);
                    ins_mem->executeUpdate();
                    ins_mem->setInt64(1, channelId);
                    ins_mem->setInt64(2, receiverId);
                    ins_mem->executeUpdate();

                    // push notification to receiver (if connected)
                    Membership m; m.channel_id = channelId; m.user_id = receiverId;
                    sendMessageOverWebSocket(std::vector<Membership>{m}, channelCreateMsg, senderName);

                    json j;
                    j["channel_id"] = channelId;
                    j["msg"] = channelCreateMsg;
                    output.push_back(j);
                } else {
                    // fetch last 10 messages for that channel (order desc)
                    std::unique_ptr<sql::PreparedStatement> q2(dbConn->prepareStatement(
                        "SELECT message_id, sender_id, channel_id, msg, created_at FROM message "
                        "WHERE channel_id = ? ORDER BY created_at DESC LIMIT 10"));
                    q2->setInt64(1, channelId);
                    std::unique_ptr<sql::ResultSet> rs2(q2->executeQuery());
                    while (rs2->next()) {
                        json jm;
                        jm["message_id"] = rs2->getInt64("message_id");
                        jm["sender_id"] = rs2->getInt64("sender_id");
                        jm["channel_id"] = rs2->getInt64("channel_id");
                        jm["msg"] = rs2->getString("msg");
                        jm["created_at"] = rs2->getString("created_at");
                        output.push_back(jm);
                    }
                }

                res.set_header("Content-Type", "application/json");
                res.write(json(output).dump());
                res.end();
            } catch (sql::SQLException &e) {
                std::cerr << "createChannel DB error: " << e.what() << "\n";
                res.code = 500;
                res.write("try again");
                res.end();
            }
        });

    // sendMessage endpoint: /message/{senderId}/{channelId}
    CROW_ROUTE(app, "/message/<int>/<int>").methods("POST"_method)(
        [](const crow::request &req, crow::response &res, int senderId, int channelId) {
            try {
                auto body = json::parse(req.body);
                std::string msg = body.value("msg", "");
                if (msg.empty()) {
                    res.code = 400;
                    res.write("Invalid request payload");
                    res.end();
                    return;
                }

                {
                    std::lock_guard<std::mutex> lk(dbMutex);
                    // insert message
                    std::unique_ptr<sql::PreparedStatement> ins(dbConn->prepareStatement(
                        "INSERT INTO message (sender_id, channel_id, msg, created_at) VALUES (?, ?, ?, ?)"));
                    using clock = std::chrono::system_clock;
                    auto now = clock::now();
                    ins->setInt64(1, senderId);
                    ins->setInt64(2, channelId);
                    ins->setString(3, msg);
                    ins->setString(4, timePointToSQL(now));
                    ins->executeUpdate();

                    // fetch members of channel except sender
                    std::unique_ptr<sql::PreparedStatement> q(dbConn->prepareStatement(
                        "SELECT membership_id, channel_id, user_id FROM membership WHERE channel_id = ?"));
                    q->setInt64(1, channelId);
                    std::unique_ptr<sql::ResultSet> rs(q->executeQuery());
                    std::vector<Membership> members;
                    while (rs->next()) {
                        long uid = rs->getInt64("user_id");
                        long mid = rs->getInt64("membership_id");
                        long cid = rs->getInt64("channel_id");
                        if (uid != senderId) {
                            Membership m; m.membership_id = mid; m.channel_id = cid; m.user_id = uid;
                            members.push_back(m);
                        }
                    }

                    // find sender username
                    std::unique_ptr<sql::PreparedStatement> q2(dbConn->prepareStatement(
                        "SELECT userName FROM user WHERE id = ?"));
                    q2->setInt64(1, senderId);
                    std::unique_ptr<sql::ResultSet> rs2(q2->executeQuery());
                    std::string senderName = "unknown";
                    if (rs2->next()) senderName = rs2->getString("userName");

                    sendMessageOverWebSocket(members, msg, senderName);
                }

                res.set_header("Content-Type", "application/json");
                res.write(json({{"msg", msg}}).dump());
                res.end();
            } catch (sql::SQLException &e) {
                std::cerr << "sendMessage DB error: " << e.what() << "\n";
                res.code = 500;
                res.write("try again");
                res.end();
            } catch (const json::exception &je) {
                res.code = 400;
                res.write("Invalid JSON");
                res.end();
            }
        });

    // status endpoint: /status (GET)
    CROW_ROUTE(app, "/status").methods("GET"_method)(
        [](const crow::request &req, crow::response &res) {
            try {
                std::lock_guard<std::mutex> lk(dbMutex);
                std::unique_ptr<sql::PreparedStatement> ps(dbConn->prepareStatement(
                    "SELECT id, userName, last_timestamp FROM user"));
                std::unique_ptr<sql::ResultSet> rs(ps->executeQuery());

                json out = json::object();
                using clock = std::chrono::system_clock;
                auto now = clock::now();
                while (rs->next()) {
                    long id = rs->getInt64("id");
                    std::string name = rs->getString("userName");
                    std::string ts = rs->getString("last_timestamp"); // "YYYY-MM-DD HH:MM:SS"
                    auto tp = parseSQLTime(ts);
                    auto diff = std::chrono::duration_cast<std::chrono::seconds>(now - tp);
                    if (diff <= std::chrono::seconds(10)) out[name] = true;
                    else out[name] = false;
                }

                res.set_header("Content-Type", "application/json");
                res.write(out.dump());
                res.end();
            } catch (sql::SQLException &e) {
                std::cerr << "status DB error: " << e.what() << "\n";
                res.code = 500;
                res.write("error from db");
                res.end();
            }
        });

    // WebSocket handler: /ws
    CROW_ROUTE(app, "/ws").websocket()
        .onopen([](crow::websocket::connection &conn) {
            std::cout << "Client connected: " << conn.get_remote_endpoint() << " id=" << conn.get_id() << "\n";
        })
        .onclose([](crow::websocket::connection &conn, const std::string &reason) {
            std::cout << "Client disconnected: id=" << conn.get_id() << " reason=" << reason << "\n";
            // remove conn from wsMap
            std::lock_guard<std::mutex> lk(wsMutex);
            for (auto it = wsMap.begin(); it != wsMap.end();) {
                auto &vec = it->second;
                vec.erase(std::remove(vec.begin(), vec.end(), &conn), vec.end());
                if (vec.empty()) it = wsMap.erase(it);
                else ++it;
            }
        })
        .onmessage([](crow::websocket::connection &conn, const std::string &data, bool is_binary) {
            // Expect heartbeat message from client: "userId,username"
            try {
                std::string s = data;
                // Trim
                s.erase(0, s.find_first_not_of(" \n\r\t"));
                s.erase(s.find_last_not_of(" \n\r\t") + 1);
                if (s.find(',') != std::string::npos) {
                    auto comma = s.find(',');
                    std::string uidStr = s.substr(0, comma);
                    std::string userName = s.substr(comma + 1);
                    long uid = std::stol(uidStr);

                    {
                        std::lock_guard<std::mutex> lk(wsMutex);
                        wsMap[uid].push_back(&conn);
                    }

                    // upsert into user table last_timestamp
                    try {
                        std::lock_guard<std::mutex> lk(dbMutex);
                        std::unique_ptr<sql::PreparedStatement> ps(dbConn->prepareStatement(
                            "INSERT INTO user (id, userName, last_timestamp) VALUES (?, ?, ?)"
                            "ON DUPLICATE KEY UPDATE last_timestamp = ?"));
                        using clock = std::chrono::system_clock;
                        auto now = clock::now();
                        std::string nowStr = timePointToSQL(now);
                        ps->setInt64(1, uid);
                        ps->setString(2, userName);
                        ps->setString(3, nowStr);
                        ps->setString(4, nowStr);
                        ps->executeUpdate();
                    } catch (sql::SQLException &e) {
                        std::cerr << "heartbeat DB upsert error: " << e.what() << "\n";
                    }
                } else {
                    // other messages can be logged
                    std::cout << "ws message: " << data << "\n";
                }
            } catch (const std::exception &e) {
                std::cerr << "ws onmessage error: " << e.what() << "\n";
            }
        });

    std::cout << "starting server on :4444\n";
    app.port(4444).multithreaded().run();
    return 0;
}
