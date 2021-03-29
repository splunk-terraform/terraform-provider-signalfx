package integration

// AzureService is a type for Azure Service names
type AzureService string

// List of AzureService
const (
	AZURE_SQL_SERVERS_ELASTICPOOLS                               AzureService = "microsoft.sql/servers/elasticpools"
	AZURE_STORAGE_STORAGEACCOUNTS                                AzureService = "microsoft.storage/storageaccounts"
	AZURE_STORAGE_STORAGEACCOUNTS_TABLESERVICES                  AzureService = "microsoft.storage/storageaccounts/tableservices"
	AZURE_STORAGE_STORAGEACCOUNTS_BLOBSERVICES                   AzureService = "microsoft.storage/storageaccounts/blobservices"
	AZURE_STORAGE_STORAGEACCOUNTS_QUEUESERVICES                  AzureService = "microsoft.storage/storageaccounts/queueservices"
	AZURE_STORAGE_STORAGEACCOUNTS_FILESERVICES                   AzureService = "microsoft.storage/storageaccounts/fileservices"
	AZURE_COMPUTE_VIRTUALMACHINESCALESETS                        AzureService = "microsoft.compute/virtualmachinescalesets"
	AZURE_COMPUTE_VIRTUALMACHINESCALESETS_VIRTUALMACHINES        AzureService = "microsoft.compute/virtualmachinescalesets/virtualmachines"
	AZURE_COMPUTE_VIRTUALMACHINES                                AzureService = "microsoft.compute/virtualmachines"
	AZURE_DEVICES                                                AzureService = "microsoft.devices"
	AZURE_DEVICES_IOTHUBS                                        AzureService = "microsoft.devices/iothubs"
	AZURE_DEVICES_ELASTICPOOLS                                   AzureService = "microsoft.devices/elasticpools"
	AZURE_DEVICES_ELASTICPOOLS_IOHUBTENANTS                      AzureService = "microsoft.devices/elasticpools/iothubtenants"
	AZURE_EVENTHUB_NAMESPACES                                    AzureService = "microsoft.eventhub/namespaces"
	AZURE_BATCH_BATCHACCOUNTS                                    AzureService = "microsoft.batch/batchaccounts"
	AZURE_SQL_SERVERS_DATABASES                                  AzureService = "microsoft.sql/servers/databases"
	AZURE_CACHE_REDIS                                            AzureService = "microsoft.cache/redis"
	AZURE_LOGIC_WORKFLOWS                                        AzureService = "microsoft.logic/workflows"
	AZURE_MICROSOFT_WEB                                          AzureService = "microsoft.web"
	AZURE_MICROSOFT_WEB_SITES                                    AzureService = "microsoft.web/sites"
	AZURE_MICROSOFT_WEB_SERVERFARMS                              AzureService = "microsoft.web/serverfarms"
	AZURE_MICROSOFT_WEB_SITES_SLOTS                              AzureService = "microsoft.web/sites/slots"
	AZURE_MICROSOFT_WEB_HOSTINGENVIRONMENTS_MULTIROLEPOOLS       AzureService = "microsoft.web/hostingenvironments/multirolepools"
	AZURE_MICROSOFT_WEB_HOSTINGENVIRONMENTS_WORKERPOOLS          AzureService = "microsoft.web/hostingenvironments/workerpools"
	AZURE_MICROSOFT_ANALYSISSERVICES_SERVERS                     AzureService = "microsoft.analysisservices/servers"
	AZURE_MICROSOFT_APIMANAGEMENT_SERVICE                        AzureService = "microsoft.apimanagement/service"
	AZURE_MICROSOFT_AUTOMATION_AUTOMATIONACCOUNTS                AzureService = "microsoft.automation/automationaccounts"
	AZURE_MICROSOFT_CLASSICCOMPUTE_VIRTUALMACHINES               AzureService = "microsoft.classiccompute/virtualmachines"
	AZURE_MICROSOFT_COGNITIVESERVICES_ACCOUNTS                   AzureService = "microsoft.cognitiveservices/accounts"
	AZURE_MICROSOFT_CUSTOMERINSIGHTS_HUBS                        AzureService = "microsoft.customerinsights/hubs"
	AZURE_MICROSOFT_DATAFACTORY                                  AzureService = "microsoft.datafactory"
	AZURE_MICROSOFT_DATAFACTORY_DATAFACTORIES                    AzureService = "microsoft.datafactory/datafactories"
	AZURE_MICROSOFT_DATAFACTORY_FACTORIES                        AzureService = "microsoft.datafactory/factories"
	AZURE_MICROSOFT_DATALAKEANALYTICS_ACCOUNTS                   AzureService = "microsoft.datalakeanalytics/accounts"
	AZURE_MICROSOFT_DATALAKESTORE_ACCOUNTS                       AzureService = "microsoft.datalakestore/accounts"
	AZURE_MICROSOFT_DBFORMYSQL_SERVERS                           AzureService = "microsoft.dbformysql/servers"
	AZURE_MICROSOFT_DBFORPOSTGRESQL_SERVERS                      AzureService = "microsoft.dbforpostgresql/servers"
	AZURE_MICROSOFT_DOCUMENTDB_DATABASEACCOUNTS                  AzureService = "microsoft.documentdb/databaseaccounts"
	AZURE_MICROSOFT_KEYVAULT_VAULTS                              AzureService = "microsoft.keyvault/vaults"
	AZURE_MICROSOFT_LOCATIONBASEDSERVICES_ACCOUNTS               AzureService = "microsoft.locationbasedservices/accounts"
	AZURE_MICROSOFT_NETWORK_LOADBALANCERS                        AzureService = "microsoft.network/loadbalancers"
	AZURE_MICROSOFT_NETWORK_PUBLICIPADDRESSES                    AzureService = "microsoft.network/publicipaddresses"
	AZURE_MICROSOFT_NETWORK_APPLICATIONGATEWAYS                  AzureService = "microsoft.network/applicationgateways"
	AZURE_MICROSOFT_NETWORK_VIRTUALNETWORKGATEWAYS               AzureService = "microsoft.network/virtualnetworkgateways"
	AZURE_MICROSOFT_NETWORK_EXPRESSROUTECIRCUITS                 AzureService = "microsoft.network/expressroutecircuits"
	AZURE_MICROSOFT_NETWORK_TRAFFICMANAGERPROFILES               AzureService = "microsoft.network/trafficmanagerprofiles"
	AZURE_MICROSOFT_NOTIFICATIONHUBS_NAMESPACES_NOTIFICATIONHUBS AzureService = "microsoft.notificationhubs/namespaces/notificationhubs"
	AZURE_MICROSOFT_POWERBIDEDICATED_CAPACITIES                  AzureService = "microsoft.powerbidedicated/capacities"
	AZURE_MICROSOFT_RELAY_NAMESPACES                             AzureService = "microsoft.relay/namespaces"
	AZURE_MICROSOFT_SEARCH_SEARCHSERVICES                        AzureService = "microsoft.search/searchservices"
	AZURE_MICROSOFT_SERVICEBUS_NAMESPACES                        AzureService = "microsoft.servicebus/namespaces"
	AZURE_MICROSOFT_SQL_SERVERS                                  AzureService = "microsoft.sql/servers"
	AZURE_MICROSOFT_STREAMANALYTICS_STREAMINGJOBS                AzureService = "microsoft.streamanalytics/streamingjobs"
	AZURE_MICROSOFT_NETWORK_DNSZONES                             AzureService = "microsoft.network/dnszones"
	AZURE_MICROSOFT_HDINSIGHT_CLUSTERS                           AzureService = "microsoft.hdinsight/clusters"
	AZURE_MICROSOFT_CONTAINERINSTANCE_CONTAINERGROUPS            AzureService = "microsoft.containerinstance/containergroups"
	AZURE_MICROSOFT_CONTAINERINSTANCE_MANAGEDCLUSTERS            AzureService = "microsoft.containerservice/managedclusters"
	AZURE_MICROSOFT_KUSTO_CLUSTERS                               AzureService = "microsoft.kusto/clusters"
	AZURE_MICROSOFT_MACHINELEARNINGSERVICES_WORKSPACES           AzureService = "microsoft.machinelearningservices/workspaces"
	AZURE_MICROSOFT_EVENTGRID_EVENTSSUBSCRIPTIONS                AzureService = "microsoft.eventgrid/eventsubscriptions"
	AZURE_MICROSOFT_EVENTGRID_EXTENSIONTOPICS                    AzureService = "microsoft.eventgrid/extensiontopics"
	AZURE_MICROSOFT_EVENTGRID_SYSTEMTOPICS                       AzureService = "microsoft.eventgrid/systemtopics"
	AZURE_MICROSOFT_EVENTGRID_TOPICS                             AzureService = "microsoft.eventgrid/topics"
	AZURE_MICROSOFT_EVENTGRID_DOMAINS                            AzureService = "microsoft.eventgrid/domains"
	AZURE_MICROSOFT_NETWORK_FRONTDOORS                           AzureService = "microsoft.network/frontdoors"
	AZURE_MICROSOFT_NETWORK_AZUREFIREWALLS                       AzureService = "microsoft.network/azurefirewalls"
	AZURE_MICROSOFT_MAPS_ACCOUNTS                                AzureService = "microsoft.maps/accounts"
	AZURE_MICROSOFT_NETWORK_NETWORKINTERFACES                    AzureService = "microsoft.network/networkinterfaces"
)

// AzureServiceNames is a list of Azure service names
var AzureServiceNames = map[string]AzureService{
	"microsoft.sql/servers/elasticpools":                        AZURE_SQL_SERVERS_ELASTICPOOLS,
	"microsoft.storage/storageaccounts":                         AZURE_STORAGE_STORAGEACCOUNTS,
	"microsoft.storage/storageaccounts/tableservices":           AZURE_STORAGE_STORAGEACCOUNTS_TABLESERVICES,
	"microsoft.storage/storageaccounts/blobservices":            AZURE_STORAGE_STORAGEACCOUNTS_BLOBSERVICES,
	"microsoft.storage/storageaccounts/queueservices":           AZURE_STORAGE_STORAGEACCOUNTS_QUEUESERVICES,
	"microsoft.storage/storageaccounts/fileservices":            AZURE_STORAGE_STORAGEACCOUNTS_FILESERVICES,
	"microsoft.compute/virtualmachinescalesets":                 AZURE_COMPUTE_VIRTUALMACHINESCALESETS,
	"microsoft.compute/virtualmachinescalesets/virtualmachines": AZURE_COMPUTE_VIRTUALMACHINESCALESETS_VIRTUALMACHINES,
	"microsoft.compute/virtualmachines":                         AZURE_COMPUTE_VIRTUALMACHINES,
	"microsoft.devices":                                         AZURE_DEVICES,
	"microsoft.devices/iothubs":                                 AZURE_DEVICES_IOTHUBS,
	"microsoft.devices/elasticpools":                            AZURE_DEVICES_ELASTICPOOLS,
	"microsoft.devices/elasticpools/iothubtenants":              AZURE_DEVICES_ELASTICPOOLS_IOHUBTENANTS,
	"microsoft.eventhub/namespaces":                             AZURE_EVENTHUB_NAMESPACES,
	"microsoft.batch/batchaccounts":                             AZURE_BATCH_BATCHACCOUNTS,
	"microsoft.sql/servers/databases":                           AZURE_SQL_SERVERS_DATABASES,
	"microsoft.cache/redis":                                     AZURE_CACHE_REDIS,
	"microsoft.logic/workflows":                                 AZURE_LOGIC_WORKFLOWS,
	"microsoft.web":                                             AZURE_MICROSOFT_WEB,
	"microsoft.web/sites":                                       AZURE_MICROSOFT_WEB_SITES,
	"microsoft.web/serverfarms":                                 AZURE_MICROSOFT_WEB_SERVERFARMS,
	"microsoft.web/sites/slots":                                 AZURE_MICROSOFT_WEB_SITES_SLOTS,
	"microsoft.web/hostingenvironments/multirolepools":          AZURE_MICROSOFT_WEB_HOSTINGENVIRONMENTS_MULTIROLEPOOLS,
	"microsoft.web/hostingenvironments/workerpools":             AZURE_MICROSOFT_WEB_HOSTINGENVIRONMENTS_WORKERPOOLS,
	"microsoft.analysisservices/servers":                        AZURE_MICROSOFT_ANALYSISSERVICES_SERVERS,
	"microsoft.apimanagement/service":                           AZURE_MICROSOFT_APIMANAGEMENT_SERVICE,
	"microsoft.automation/automationaccounts":                   AZURE_MICROSOFT_AUTOMATION_AUTOMATIONACCOUNTS,
	"microsoft.classiccompute/virtualmachines":                  AZURE_MICROSOFT_CLASSICCOMPUTE_VIRTUALMACHINES,
	"microsoft.cognitiveservices/accounts":                      AZURE_MICROSOFT_COGNITIVESERVICES_ACCOUNTS,
	"microsoft.customerinsights/hubs":                           AZURE_MICROSOFT_CUSTOMERINSIGHTS_HUBS,
	"microsoft.datafactory":                                     AZURE_MICROSOFT_DATAFACTORY,
	"microsoft.datafactory/datafactories":                       AZURE_MICROSOFT_DATAFACTORY_DATAFACTORIES,
	"microsoft.datafactory/factories":                           AZURE_MICROSOFT_DATAFACTORY_FACTORIES,
	"microsoft.datalakeanalytics/accounts":                      AZURE_MICROSOFT_DATALAKEANALYTICS_ACCOUNTS,
	"microsoft.datalakestore/accounts":                          AZURE_MICROSOFT_DATALAKESTORE_ACCOUNTS,
	"microsoft.dbformysql/servers":                              AZURE_MICROSOFT_DBFORMYSQL_SERVERS,
	"microsoft.dbforpostgresql/servers":                         AZURE_MICROSOFT_DBFORPOSTGRESQL_SERVERS,
	"microsoft.documentdb/databaseaccounts":                     AZURE_MICROSOFT_DOCUMENTDB_DATABASEACCOUNTS,
	"microsoft.keyvault/vaults":                                 AZURE_MICROSOFT_KEYVAULT_VAULTS,
	"microsoft.locationbasedservices/accounts":                  AZURE_MICROSOFT_LOCATIONBASEDSERVICES_ACCOUNTS,
	"microsoft.network/loadbalancers":                           AZURE_MICROSOFT_NETWORK_LOADBALANCERS,
	"microsoft.network/publicipaddresses":                       AZURE_MICROSOFT_NETWORK_PUBLICIPADDRESSES,
	"microsoft.network/applicationgateways":                     AZURE_MICROSOFT_NETWORK_APPLICATIONGATEWAYS,
	"microsoft.network/virtualnetworkgateways":                  AZURE_MICROSOFT_NETWORK_VIRTUALNETWORKGATEWAYS,
	"microsoft.network/expressroutecircuits":                    AZURE_MICROSOFT_NETWORK_EXPRESSROUTECIRCUITS,
	"microsoft.network/trafficmanagerprofiles":                  AZURE_MICROSOFT_NETWORK_TRAFFICMANAGERPROFILES,
	"microsoft.notificationhubs/namespaces/notificationhubs":    AZURE_MICROSOFT_NOTIFICATIONHUBS_NAMESPACES_NOTIFICATIONHUBS,
	"microsoft.powerbidedicated/capacities":                     AZURE_MICROSOFT_POWERBIDEDICATED_CAPACITIES,
	"microsoft.relay/namespaces":                                AZURE_MICROSOFT_RELAY_NAMESPACES,
	"microsoft.search/searchservices":                           AZURE_MICROSOFT_SEARCH_SEARCHSERVICES,
	"microsoft.servicebus/namespaces":                           AZURE_MICROSOFT_SERVICEBUS_NAMESPACES,
	"microsoft.sql/servers":                                     AZURE_MICROSOFT_SQL_SERVERS,
	"microsoft.streamanalytics/streamingjobs":                   AZURE_MICROSOFT_STREAMANALYTICS_STREAMINGJOBS,
	"microsoft.network/dnszones":                                AZURE_MICROSOFT_NETWORK_DNSZONES,
	"microsoft.hdinsight/clusters":                              AZURE_MICROSOFT_HDINSIGHT_CLUSTERS,
	"microsoft.containerinstance/containergroups":               AZURE_MICROSOFT_CONTAINERINSTANCE_CONTAINERGROUPS,
	"microsoft.containerservice/managedclusters":                AZURE_MICROSOFT_CONTAINERINSTANCE_MANAGEDCLUSTERS,
	"microsoft.kusto/clusters":                                  AZURE_MICROSOFT_KUSTO_CLUSTERS,
	"microsoft.machinelearningservices/workspaces":              AZURE_MICROSOFT_MACHINELEARNINGSERVICES_WORKSPACES,
	"microsoft.eventgrid/eventsubscriptions":                    AZURE_MICROSOFT_EVENTGRID_EVENTSSUBSCRIPTIONS,
	"microsoft.eventgrid/extensiontopics":                       AZURE_MICROSOFT_EVENTGRID_EXTENSIONTOPICS,
	"microsoft.eventgrid/systemtopics":                          AZURE_MICROSOFT_EVENTGRID_SYSTEMTOPICS,
	"microsoft.eventgrid/topics":                                AZURE_MICROSOFT_EVENTGRID_TOPICS,
	"microsoft.eventgrid/domains":                               AZURE_MICROSOFT_EVENTGRID_DOMAINS,
	"microsoft.network/frontdoors":                              AZURE_MICROSOFT_NETWORK_FRONTDOORS,
	"microsoft.network/azurefirewalls":                          AZURE_MICROSOFT_NETWORK_AZUREFIREWALLS,
	"microsoft.maps/accounts":                                   AZURE_MICROSOFT_MAPS_ACCOUNTS,
	"microsoft.network/networkinterfaces":                       AZURE_MICROSOFT_NETWORK_NETWORKINTERFACES,
}
