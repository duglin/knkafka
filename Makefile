all: .knkafkaservice .knkafkaconsole admin deployKafka deploy

.knkafkaservice: service.go
	docker build -f Dockerfile.service -t duglin/knkafkaservice .
	docker push duglin/knkafkaservice
	touch .knkafkaservice

.knkafkaconsole: console.go
	docker build -f Dockerfile.console -t duglin/knkafkaconsole .
	docker push duglin/knkafkaconsole
	touch .knkafkaconsole

deployKafka:
	kubectl apply -f zookeeper.yaml
	kubectl apply -f kafka-service.yaml
	sleep 5
	cat kafka-deployment.yaml | \
	  sed "s/MY_IP/$$(kubectl get service/kafka-service -o yaml | grep ip: | sed "s/.*: *//")/" | \
	  sed "s/MY_PORT/$$(kubectl get service/kafka-service -o yaml | grep nodePort: | sed "s/.*: *//")/" | \
	  kubectl apply -f -
	kubectl describe service/kafka-service | grep Ingress | sed "s/.*: *//" > .server

deploy: .knkafkaservice .knkafkaconsole admin
	kubectl apply -f knservices.yaml
	# kubectl apply -f source.yaml
	# kn service create knkafkaservice --image duglin/knkafkaservice
	# kn service create knkafkaconsole --image duglin/knkafkaconsole \
		# --min-scale=1 --max-scale=1

admin: admin.go
	go build -o admin admin.go

watch: admin
	watch -n 5 'curl -s $(shell kubectl get ksvc/knkafkaconsole -o jsonpath={.status.url}) ; echo ; kubectl get pods ;  echo ; ./admin | grep size'

clean:
	-kubectl delete -f source.yaml > /dev/null 2>&1
	-kubectl delete -f knservices.yaml
	# -kn service delete knkafkaservice > /dev/null 2>&1
	# -kn service delete knkafkaconsole > /dev/null 2>&1
	-rm admin client
	-kubectl delete -f kafka-deployment.yaml
	-kubectl delete -f kafka-service.yaml
	-kubectl delete -f zookeeper.yaml
	-rm -f .knkafkaconsole .knkafkaservice .server
