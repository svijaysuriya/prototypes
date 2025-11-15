high availability (reliable ) => replication
low latency ( fast )

user -> lb -> redirects to available backend server!

{
    backendServer: []
    algorithm: ""
}

add a header in the response so that i will be able to see from backend server the response came /
or log it in the lb server!