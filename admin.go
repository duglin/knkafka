package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Shopify/sarama"
)

type topicsMD []*sarama.TopicMetadata

func (t topicsMD) Len() int           { return len(t) }
func (t topicsMD) Less(i, j int) bool { return t[i].Name < t[j].Name }
func (t topicsMD) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }

func main() {
	server := ""

	if tmp := os.Getenv("SERVER"); tmp != "" {
		server = tmp
	} else {
		buf, err := ioutil.ReadFile(".server")
		if err == nil && len(buf) > 0 {
			server = string(buf)
		}
	}

	if server == "" {
		fmt.Printf("Missing server - set SERVER or .server file\n")
		os.Exit(1)
	}

	if strings.Index(server, ":") < 0 {
		server += ":9092"
	}

	config := sarama.NewConfig()
	config.Version = sarama.V1_0_0_0

	if len(os.Args) == 1 || os.Args[1] == "list" {
		client, err := sarama.NewClient([]string{server}, config)
		if err != nil {
			fmt.Printf("Client: %s\n", err)
			os.Exit(1)
		}
		defer client.Close()

		topics, err := client.Topics()
		if err != nil {
			fmt.Printf("Topics: %s\n", err)
			os.Exit(1)
		}
		sort.Strings(topics)

		for _, topic := range topics {
			fmt.Printf("Topic: %s", topic)

			partitions, err := client.Partitions(topic)
			chars := len(strconv.Itoa(len(partitions)))
			if err != nil {
				fmt.Printf("Partitions: %s\n", err)
				os.Exit(1)
			}
			sizes := map[int32]int64{}
			size := int64(0)

			for _, p := range partitions {
				off, err := client.GetOffset(topic, p, -1)
				if err != nil {
					fmt.Printf("Offset: %s\n", err)
					os.Exit(1)
				}
				sizes[p] = off
				size += off
			}
			fmt.Printf("  - Partitions: %d (size: %d)\n", len(partitions), size)
			for _, p := range partitions {
				if topic == "__consumer_offsets" && sizes[p] == 0 {
					continue
				}
				fstr := fmt.Sprintf("    - %%%dd: %%d\n", chars)
				fmt.Printf(fstr, p, sizes[p])
			}
		}
	} else if len(os.Args) == 1 || os.Args[1] == "list" {
		admin, err := sarama.NewClusterAdmin([]string{server}, config)
		if err != nil {
			fmt.Printf("Admin: %s\n", err)
			os.Exit(1)
		}
		defer admin.Close()

		client, err := sarama.NewClient([]string{server}, config)
		if err != nil {
			fmt.Printf("Client: %s\n", err)
			os.Exit(1)
		}
		defer client.Close()

		offMgr, err := sarama.NewOffsetManagerFromClient("knative-group", client)
		if err != nil {
			fmt.Printf("OffMgr: %s\n", err)
			os.Exit(1)
		}
		defer offMgr.Close()

		topics, err := admin.ListTopics()
		if err != nil {
			fmt.Printf("Topics: %s\n", err)
			os.Exit(1)
		}

		topicNames := []string{}

		for name, _ := range topics {
			topicNames = append(topicNames, name)
		}

		topicsMetadata, err := admin.DescribeTopics(topicNames)
		if err != nil {
			fmt.Printf("TopicsMetadata: %s\n", err)
			os.Exit(1)
		}

		t := topicsMD(topicsMetadata)
		sort.Sort(t)

		for _, topicMetadata := range t {
			fmt.Printf("Topic: %-30s", topicMetadata.Name)
			fmt.Printf("  - IsInternal: %5v", topicMetadata.IsInternal)
			fmt.Printf("  - Partitions: %d\n", len(topicMetadata.Partitions))
			for i := 0; i < len(topicMetadata.Partitions); i++ {
				POM, err := offMgr.ManagePartition(topicMetadata.Name, int32(i))
				if err != nil {
					fmt.Printf("POM: %s\n", err)
					os.Exit(1)
				}
				len, str := POM.NextOffset()
				fmt.Printf("    - %d: %d %-50.50s\n", i, len, str)
			}

		}
	} else if len(os.Args) > 1 && os.Args[1] == "del-topic" {
		if len(os.Args) < 3 {
			fmt.Printf("Usage: %s %s TOPIC ...\n", os.Args[0], os.Args[1])
			os.Exit(1)
		}

		admin, err := sarama.NewClusterAdmin([]string{server}, config)
		if err != nil {
			fmt.Printf("Admin: %s\n", err)
			os.Exit(1)
		}
		defer admin.Close()

		for i := 2; i < len(os.Args); i++ {
			topic := os.Args[i]
			err = admin.DeleteTopic(topic)
			if err != nil {
				fmt.Printf("Error deleteing topic %q: %s\n", topic, err)
				os.Exit(1)
			}
			// fmt.Printf("Deleted: %s\n", topic)
		}
	} else if len(os.Args) > 1 && os.Args[1] == "add-topic" {
		var err error

		if len(os.Args) != 3 && len(os.Args) != 4 {
			fmt.Printf("Usage: %s %s TOPIC [NUM_PARTITIONS]\n",
				os.Args[0], os.Args[1])
			os.Exit(1)
		}

		num := 1
		if len(os.Args) == 4 {
			num, err = strconv.Atoi(os.Args[3])
			if err != nil {
				fmt.Printf("%q isn't a valid number of partitions", os.Args[3])
			}
		}

		admin, err := sarama.NewClusterAdmin([]string{server}, config)
		if err != nil {
			fmt.Printf("Admin: %s\n", err)
			os.Exit(1)
		}
		defer admin.Close()

		topic := os.Args[2]
		topicDetail := sarama.TopicDetail{
			NumPartitions:     int32(num),
			ReplicationFactor: 1,
		}

		err = admin.CreateTopic(topic, &topicDetail, false)
		if err != nil {
			fmt.Printf("Error creating topic %q: %s\n", topic, err)
			os.Exit(1)
		}
	} else if len(os.Args) > 1 && os.Args[1] == "load" {
		var err error

		if len(os.Args) != 3 && len(os.Args) != 4 {
			fmt.Printf("Usage: %s %s TOPIC NUM_MSGS\n",
				os.Args[0], os.Args[1])
			os.Exit(1)
		}

		num := 1
		if len(os.Args) == 4 {
			num, err = strconv.Atoi(os.Args[3])
			if err != nil {
				fmt.Printf("%q isn't a valid number of messages", os.Args[3])
			}
		}

		config.Producer.Return.Successes = true
		client, err := sarama.NewSyncProducer([]string{server}, config)
		if err != nil {
			fmt.Printf("Client: %s\n", err)
			os.Exit(1)
		}
		defer client.Close()

		topic := os.Args[2]
		count := 0
		countMutex := sync.Mutex{}
		wg := sync.WaitGroup{}

		for i := 0; i < num; i++ {
			if count >= 200 {
				time.Sleep(100 * time.Millisecond)
				i--
				continue
			}

			countMutex.Lock()
			count++
			wg.Add(1)
			countMutex.Unlock()
			go func() {
				text := fmt.Sprintf("%s %d", topic, i) // TOPIC i
				msg := sarama.ProducerMessage{
					Topic: topic,
					Value: sarama.StringEncoder(text),
				}
				_, _, err = client.SendMessage(&msg)
				if err != nil {
					fmt.Printf("Error sending msg: %s\n", err)
					os.Exit(1)
				}
				countMutex.Lock()
				count--
				wg.Done()
				countMutex.Unlock()
			}()
		}
		wg.Wait()
		if count != 0 {
			fmt.Printf("Something is wrong: %d\n", count)
		}
	} else {
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
