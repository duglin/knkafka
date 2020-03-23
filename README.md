# knkafka
Demo of Knative using Kafka

Run: `make` to deploy Kafka and the Knative
Services: `knkafkaconsole` and `knkafkaservice`

To do the demo:
- (optional) run `make watch` in one window
- run `./doit` in another to actually do the demo


Run: `make clean` to take it all down

Technically, all a user (consumer of the messages) would need to do is:
- write their kn service (`service.go`) & build an IMAGE
- deploy it: `kn service create knkafkaservice --image IMAGE`
- create a Kafka event source to get the events - see: `source.yaml`

The rest of the stuff in this repo is either kafka setup stuff or just noise.

## Files:

Misc:
- Makefile
- README.md
- admin.go : tool to help me do Kafka stuff, like create topics and load records
- doit : runner script to actually run the demo (cleans, loads, runs)
- kafka-deployment.yaml : Kafka itself
- kafka-service.yaml : More Kafka stuff
- zookeeper.yaml : ZooKeeper is used by Kafka

User created stuff:
- Dockerfile.console : For building the console service
- Dockerfile.service : for building the service that'll process the records
- console.go : simple tool to report status of the record processing
- knservices.yaml : yaml to deploy the console and record processing services
- service.go : code for the record processing service
- source.yaml : yaml to deploy the Kafka Event Source. This connects our
  service up to the Kafka topics
