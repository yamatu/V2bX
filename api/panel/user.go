package panel

import (
	"fmt"

	"github.com/goccy/go-json"
	"github.com/go-resty/resty/v2" // 确保你使用的是 resty 库
)

// OnlineUser 结构体表示在线用户
type OnlineUser struct {
	UID int
	IP  string
}

// UserInfo 结构体表示用户信息
type UserInfo struct {
	Id          int    `json:"id"`
	Uuid        string `json:"uuid"`
	SpeedLimit  int    `json:"speed_limit"`
	DeviceLimit int    `json:"device_limit"`
}

// UserListBody 结构体用于反序列化用户列表的响应
type UserListBody struct {
	Users []UserInfo `json:"users"`
}

// AliveMap 结构体用于存储在线用户的状态
type AliveMap struct {
	Alive map[int]int `json:"alive"`
}

// Client 结构体表示用于与 API 服务器通信的客户端
type Client struct {
	baseURL   string
	client    *resty.Client
	userEtag  string
	AliveMap  *AliveMap
}

// NewClient 用于初始化 Client 并支持指定主机和端口
func NewClient(host string, port int) *Client {
	baseURL := fmt.Sprintf("http://%s:%d", host, port)
	return &Client{
		baseURL: baseURL,
		client:  resty.New(),
	}
}

// GetUserList 从 v2board 获取用户列表
func (c *Client) GetUserList() ([]UserInfo, error) {
	path := fmt.Sprintf("%s/api/v1/server/UniProxy/user", c.baseURL)
	r, err := c.client.R().
		SetHeader("If-None-Match", c.userEtag).
		ForceContentType("application/json").
		Get(path)

	if r == nil || r.RawResponse == nil {
		return nil, fmt.Errorf("received nil response or raw response")
	}
	defer r.RawResponse.Body.Close()

	if r.StatusCode() == 304 {
		return nil, nil
	}

	if err = c.checkResponse(r, path, err); err != nil {
		return nil, err
	}
	userlist := &UserListBody{}
	if err := json.Unmarshal(r.Body(), userlist); err != nil {
		return nil, fmt.Errorf("unmarshal user list error: %w", err)
	}
	c.userEtag = r.Header().Get("ETag")
	return userlist.Users, nil
}

// GetUserAlive 获取用户的在线 IP 数量
func (c *Client) GetUserAlive() (map[int]int, error) {
	path := fmt.Sprintf("%s/api/v1/server/UniProxy/alivelist", c.baseURL)
	r, err := c.client.R().
		ForceContentType("application/json").
		Get(path)

	if r == nil || r.RawResponse == nil {
		return nil, fmt.Errorf("received nil response or raw response")
	}
	defer r.RawResponse.Body.Close()

	c.AliveMap = &AliveMap{}
	if err != nil || r.StatusCode() >= 399 {
		c.AliveMap.Alive = make(map[int]int)
		return c.AliveMap.Alive, nil
	}

	if err := json.Unmarshal(r.Body(), c.AliveMap); err != nil {
		return nil, fmt.Errorf("unmarshal user alive list error: %s", err)
	}

	return c.AliveMap.Alive, nil
}

// UserTraffic 结构体表示用户的流量信息
type UserTraffic struct {
	UID      int
	Upload   int64
	Download int64
}

// ReportUserTraffic 上报用户的流量信息
func (c *Client) ReportUserTraffic(userTraffic []UserTraffic) error {
	data := make(map[int][]int64, len(userTraffic))
	for i := range userTraffic {
		data[userTraffic[i].UID] = []int64{userTraffic[i].Upload, userTraffic[i].Download}
	}
	path := fmt.Sprintf("%s/api/v1/server/UniProxy/push", c.baseURL)
	r, err := c.client.R().
		SetBody(data).
		ForceContentType("application/json").
		Post(path)

	err = c.checkResponse(r, path, err)
	if err != nil {
		return err
	}
	return nil
}

// ReportNodeOnlineUsers 上报节点的在线用户信息
func (c *Client) ReportNodeOnlineUsers(data *map[int][]string) error {
	path := fmt.Sprintf("%s/api/v1/server/UniProxy/alive", c.baseURL)
	r, err := c.client.R().
		SetBody(data).
		ForceContentType("application/json").
		Post(path)

	err = c.checkResponse(r, path, err)
	if err != nil {
		return nil
	}

	return nil
}

// checkResponse 检查 API 请求的响应状态
func (c *Client) checkResponse(r *resty.Response, path string, err error) error {
	if err != nil {
		return err
	}

	if r.StatusCode() >= 400 {
		return fmt.Errorf("error from %s: %s", path, r.String())
	}

	return nil
}
