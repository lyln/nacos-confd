package nacos

import (
	"github.com/kelseyhightower/confd/log"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"github.com/nacos-group/nacos-sdk-go/vo"

	"fmt"
	"net/url"
	"strconv"
	"strings"
)

var replacer = strings.NewReplacer("/", ".")

type Client struct {
	configClient  config_client.IConfigClient
	namingClient  naming_client.INamingClient
	group         string
	namespace     string
	accessKey     string
	secretKey     string
	channel       chan int
	count         int
	NacosUsername string
	NacosPassword string
}

func NewNacosClient(nodes []string, group string, config constant.ClientConfig) (client *Client, err error) {
	var configClient config_client.IConfigClient
	servers := []constant.ServerConfig{}
	for _, key := range nodes {
		nacosUrl, _ := url.Parse(key)

		port, _ := strconv.Atoi(nacosUrl.Port())
		servers = append(servers, constant.ServerConfig{
			IpAddr: nacosUrl.Hostname(),
			Port:   uint64(port),
		})
	}

	if len(strings.TrimSpace(group)) == 0 {
		group = "DEFAULT_GROUP"
	}

	log.Info(fmt.Sprintf("endpoint=%s, namespace=%s, group=%s, accessKey=%s, secretKey=%s, openKMS=%d, regionId=%s, NacosUsername=%s, NacosPassword=%s", config.Endpoint, config.NamespaceId, group, config.AccessKey, config.SecretKey, config.OpenKMS, config.RegionId, config.NacosUsername, config.NacosPassword))

	configClient, err = clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": servers,
		"clientConfig": constant.ClientConfig{
			TimeoutMs:           10000,
			ListenInterval:      20000,
			NotLoadCacheAtStart: true,
			NamespaceId:         config.NamespaceId,
			AccessKey:           config.AccessKey,
			SecretKey:           config.SecretKey,
			Endpoint:            config.Endpoint,
			OpenKMS:             config.OpenKMS,
			RegionId:            config.RegionId,
			NacosUsername:       config.NacosUsername,
			NacosPassword:       config.NacosPassword,
		},
	})

	namingClient, _ := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": servers,
		"clientConfig": constant.ClientConfig{
			TimeoutMs:           10000,
			ListenInterval:      20000,
			NotLoadCacheAtStart: true,
			NamespaceId:         config.NamespaceId,
			AccessKey:           config.AccessKey,
			SecretKey:           config.SecretKey,
			Endpoint:            config.Endpoint,
		},
	})

	client = &Client{configClient, namingClient, group, config.NamespaceId, config.AccessKey, config.SecretKey, make(chan int, 10), 0, config.NacosUsername, config.NacosPassword}

	return
}

func (client *Client) GetValues(keys []string) (map[string]string, error) {
	vars := make(map[string]string)
	for _, key := range keys {
		k := strings.TrimPrefix(key, "/")
		k = replacer.Replace(k)

		if strings.HasPrefix(k, "naming.") {
			instances, err := client.namingClient.SelectAllInstances(vo.SelectAllInstancesParam{
				ServiceName: k,
				GroupName:   client.group,
				//HealthyOnly: true,
			})

			log.Info(fmt.Sprintf("key=%s, value=%s", key, instances))
			if err == nil {
				vars[key] = utils.ToJsonString(instances)
			}
		} else {
			resp, err := client.configClient.GetConfig(vo.ConfigParam{
				DataId:        k,
				Group:         client.group,
				NacosUsername: client.NacosUsername,
				NacosPassword: client.NacosPassword,
			})
			log.Info(fmt.Sprintf("key=%s, value=%s", key, resp))

			if err == nil {
				vars[key] = resp
			}
		}
	}

	return vars, nil
}

func (client *Client) WatchPrefix(prefix string, keys []string, waitIndex uint64, stopChan chan bool) (uint64, error) {
	// return something > 0 to trigger a key retrieval from the store
	if waitIndex == 0 {
		client.count++
		for _, key := range keys {
			k := strings.TrimPrefix(key, "/")
			k = replacer.Replace(k)

			if strings.HasPrefix(k, "naming.") {
				client.namingClient.Subscribe(&vo.SubscribeParam{
					ServiceName: k,
					GroupName:   client.group,
					SubscribeCallback: func(services []model.SubscribeService, err error) {
						log.Info(fmt.Sprintf("\n\n callback return services:%s \n\n", utils.ToJsonString(services)))
						for i := 0; i < client.count; i++ {
							client.channel <- 1
						}
					},
				})
			} else {
				log.Debug("WatchPrefix is action ")
				err := client.configClient.ListenConfig(vo.ConfigParam{
					DataId: k,
					Group:  client.group,
					OnChange: func(namespace, group, dataId, username, password, data string) {
						log.Debug(fmt.Sprintf("config namespace=%s, dataId=%s, group=%s, username=%s ,password=%s has changed", namespace, dataId, group, username, password))
						for i := 0; i < client.count; i++ {
							client.channel <- 1
						}
					},
					NacosUsername: client.NacosUsername,
					NacosPassword: client.NacosPassword,
				})

				if err != nil {
					return 0, err
				}
			}
		}

		return 1, nil
	}

	select {
	case <-client.channel:
		return waitIndex, nil

	}

	return waitIndex, nil
}
