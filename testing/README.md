terminal 1:
-----------
`docker compose up`

terminal 2 (after terminal 1 commands are done):
-----------
```
docker exec -it testing_nats-tools_1 sh`

# send message:
nats req HL7.MESSAGES <message content>
```
