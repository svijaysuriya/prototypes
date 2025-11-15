entities

user, group, messages, broadcasting

user:
user_id
user_name

channel
channel_id
channel_type(group, messages, broadcasting)

message:
content
from_user_id
channel_id

members
name
channel_id
userIds[]

arpit's

messages
id
sender_id
message
created_at
receiver_id

select * from messages from sender_id=B and receiver_id=A => here mssg sent by A won't be visible

select * from messages from sender_id in (B,A) and receiver_id in (A,B)

here OR query is expensive ( intersection of two union subsets!! )
also for channel(announcements) we need to have extra column channel_id

messages
id
sender_id
message
created_at
receiver_id
channel_id

is there are fundamental difference between DM and channel? 
channel consists of multiple users? so what DM is a channel with two users

messages                channel
id                      channel_id
sender_id               channel_type(DM/group/channel)
message                 name
created_at              user_ids
channel_id

now when A want to see messages from B
select * from messages where channel_id = (channel between a and b)

why design this way?
easier for us to elevate the group to channel type
also we could add easily add extensiblility, if some particular channel type is introduced, message need not worry about it!!


messages: large amount of data => we need to shard it => partition key for it?
how our query pattern will look like, whenever we get into channel or dm we need to list their messages => channel_id could be a good paritition key

don't think about search usecase( searching of message ) -> our most used query would be reading the messages in the channel and not search!

pagination should be handled by REST HTTP API / websocket?
REST HTTP API since it belongs to request response paradigm it suits best, websocket is there to elevate the user experience! ( realtime communication )

when A sends msg to B?
    to use HTTP API or websokcet? http api to post msg into db
    what happens if msg over WS to B failed? nothing to worry(data presisted), will be miss UX but user click channel it will eventually get it

    send_msg(u,m,ch_id):
        save_to_db(u,m,ch_id)
        notify_wsserver()
    
    def notify_wsserver():
        users = get_user(ch_id) // get_membership
        for each user in users:
            if user != sender:
                send msg to user through WS
        



create table channel(
    channel_id int auto_increment primary key,
    channel_type varchar(25),
    channel_name varchar(25)
);

create table message(
    message_id int auto_increment primary key,
    sender_id int,
    channel_id int,
    msg text,
    created_at timestamp,
    foreign key (channel_id) references channel(channel_id),
    foreign key (sender_id) references user(id)
);

create table membership(
    membership_id int auto_increment primary key,
    channel_id int,
    user_id int,
    foreign key (channel_id) references channel(channel_id),
    foreign key (user_id) references user(id)
);

drop table membership;
drop table message;
drop table channel;