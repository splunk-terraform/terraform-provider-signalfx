package integration

// AzureService is a type for Azure Service names
type AzureService string

// List of AzureService
const (
	AZURE_SQL_SERVERS_ELASTICPOOLS                        AzureService = "microsoft.sql/servers/elasticpools"
	AZURE_STORAGE_STORAGEACCOUNTS                         AzureService = "microsoft.storage/storageaccounts"
	AZURE_STORAGE_STORAGEACCOUNTSSERVICES_TABLESERVICES   AzureService = "microsoft.storage/storageaccountsservices/tableservices"
	AZURE_STORAGE_STORAGEACCOUNTSSERVICES_BLOBSERVICES    AzureService = "microsoft.storage/storageaccountsservices/blobservices"
	AZURE_STORAGE_STORAGEACCOUNTS_QUEUESERVICES           AzureService = "microsoft.storage/storageaccounts/queueservices"
	AZURE_STORAGE_STORAGEACCOUNTS_FILESERVICES            AzureService = "microsoft.storage/storageaccounts/fileservices"
	AZURE_COMPUTE_VIRTUALMACHINESCALESETS                 AzureService = "microsoft.compute/virtualmachinescalesets"
	AZURE_COMPUTE_VIRTUALMACHINESCALESETS_VIRTUALMACHINES AzureService = "microsoft.compute/virtualmachinescalesets/virtualmachines"
	AZURE_COMPUTE_VIRTUALMACHINES                         AzureService = "microsoft.compute/virtualmachines"
	AZURE_DEVICES_IOTHUBS                                 AzureService = "microsoft.devices/iothubs"
	AZURE_EVENTHUB_NAMESPACES                             AzureService = "microsoft.eventHub/namespaces"
	AZURE_BATCH_BATCHACCOUNTS                             AzureService = "microsoft.batch/batchaccounts"
	AZURE_SQL_SERVERS_DATABASES                           AzureService = "microsoft.sql/servers/databases"
	AZURE_CACHE_REDIS                                     AzureService = "microsoft.cache/redis"
	AZURE_LOGIC_WORKFLOWS                                 AzureService = "microsoft.logic/workflows"
)

// AzureServiceNames is a list of Azure service names
var AzureServiceNames = map[string]AzureService{
	"microsoft.sql/servers/elasticpools":                        AZURE_SQL_SERVERS_ELASTICPOOLS,
	"microsoft.storage/storageaccounts":                         AZURE_STORAGE_STORAGEACCOUNTS,
	"microsoft.storage/storageaccountsservices/tableservices":   AZURE_STORAGE_STORAGEACCOUNTSSERVICES_TABLESERVICES,
	"microsoft.storage/storageaccountsservices/blobservices":    AZURE_STORAGE_STORAGEACCOUNTSSERVICES_BLOBSERVICES,
	"microsoft.storage/storageaccounts/queueservices":           AZURE_STORAGE_STORAGEACCOUNTS_QUEUESERVICES,
	"microsoft.storage/storageaccounts/fileservices":            AZURE_STORAGE_STORAGEACCOUNTS_FILESERVICES,
	"microsoft.compute/virtualmachinescalesets":                 AZURE_COMPUTE_VIRTUALMACHINESCALESETS,
	"microsoft.compute/virtualmachinescalesets/virtualmachines": AZURE_COMPUTE_VIRTUALMACHINESCALESETS_VIRTUALMACHINES,
	"microsoft.compute/virtualmachines":                         AZURE_COMPUTE_VIRTUALMACHINES,
	"microsoft.devices/iothubs":                                 AZURE_DEVICES_IOTHUBS,
	"microsoft.eventHub/namespaces":                             AZURE_EVENTHUB_NAMESPACES,
	"microsoft.batch/batchaccounts":                             AZURE_BATCH_BATCHACCOUNTS,
	"microsoft.sql/servers/databases":                           AZURE_SQL_SERVERS_DATABASES,
	"microsoft.cache/redis":                                     AZURE_CACHE_REDIS,
	"microsoft.logic/workflows":                                 AZURE_LOGIC_WORKFLOWS,
}
