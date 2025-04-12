package etcd

const (
	NameKeyPrefix = "dfs/datanode/nodes/"
	NameKeyFormat = "dfs/datanode/nodes/%s"
)

type NodeInfo struct {
	ID       string `json:"id"`
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
	Location string `json:"location"`
}
