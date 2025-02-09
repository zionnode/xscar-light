package xscar

import (
	"context"
	"log"
	"net/http"
	"time"

	v2proxyman "github.com/xtls/xray-core/app/proxyman/command"
	v2stats "github.com/xtls/xray-core/app/stats/command"
	"google.golang.org/grpc"
)

var API_ENDPOINT string
var GRPC_ENDPOINT string

type UserConfig struct {
	UserId  int    `json:"user_id"`
	Email   string `json:"email"`
	UUID    string `json:"uuid"`
	AlterId uint32 `json:"alter_id"`
	Level   uint32 `json:"level"`
	Enable  bool   `json:"enable"`
}
type UserTraffic struct {
	UserId          int   `json:"user_id"`
	DownloadTraffic int64 `json:"dt"`
	UploadTraffic   int64 `json:"ut"`
}

type syncReq struct {
	UserTraffics []*UserTraffic `json:"user_traffics"`
}

type syncResp struct {
	Configs []*UserConfig `json:"configs"`
	Tags    []string      `json:"tags"`
}

func SyncTask(up *UserPool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, GRPC_ENDPOINT, grpc.WithInsecure(), grpc.WithBlock())

	if err != nil {
		log.Printf("[WARNING]: GRPC连接失败,请检查V2ray是否运行并开放对应grpc端口 当前GRPC地址: %v 错误信息: %v", GRPC_ENDPOINT, err.Error())
		return
	}
	defer conn.Close()

	proxymanClient := v2proxyman.NewHandlerServiceClient(conn)
	statClient := v2stats.NewStatsServiceClient(conn)
	httpClient := &http.Client{Timeout: 15 * time.Second}

	resp := syncResp{}
	err = getJson(httpClient, API_ENDPOINT, &resp)
	if err != nil {
		log.Printf("[WARNING]: API连接失败,请检查API地址 当前地址: %v 错误信息:%v", API_ENDPOINT, err.Error())
		return
	}

	// 为每个 tag 初始化或更新用户配置
	for _, tag := range resp.Tags {
		initOrUpdateUser(up, proxymanClient, &resp, tag)
	}

	syncUserTrafficToServer(up, statClient, httpClient)
}

func initOrUpdateUser(up *UserPool, c v2proxyman.HandlerServiceClient, sr *syncResp, tag string) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[INFO] Call initOrUpdateUser for tag:", tag)

	syncUserMap := make(map[string]bool)

	for _, cfg := range sr.Configs {
		syncUserMap[cfg.Email] = true
		user, _ := up.GetUserByEmail(cfg.Email)
		if user == nil {
			newUser, err := up.CreateUser(cfg.UserId, cfg.Email, cfg.UUID, cfg.Level, cfg.AlterId, cfg.Enable)
			if err != nil {
				log.Fatalln(err)
			}
			if newUser.Enable {
				AddInboundUser(c, tag, newUser)
			}
		} else {
			if user.Enable != cfg.Enable {
				user.setEnable(cfg.Enable)
			}

			if user.UUID != cfg.UUID {
				log.Printf("[INFO] user: %s 更换了uuid old: %s new: %s", user.Email, user.UUID, cfg.UUID)
				RemoveInboundUser(c, tag, user)
				user.setUUID(cfg.UUID)
				AddInboundUser(c, tag, user)
			}

			if !user.Enable && user.running {
				RemoveInboundUser(c, tag, user)
			}

			if user.Enable && !user.running {
				AddInboundUser(c, tag, user)
			}
		}
	}

	for _, user := range up.GetAllUsers() {
		if _, ok := syncUserMap[user.Email]; !ok {
			RemoveInboundUser(c, tag, user)
			up.RemoveUserByEmail(user.Email)
		}
	}
}

func syncUserTrafficToServer(up *UserPool, c v2stats.StatsServiceClient, hc *http.Client) {
	GetAndResetUserTraffic(c, up)

	tfs := make([]*UserTraffic, 0, up.GetUsersNum())
	for _, user := range up.GetAllUsers() {
		tf := user.DownloadTraffic + user.UploadTraffic
		if tf > 0 {
			log.Printf("[INFO] User: %v Now Used Total Traffic: %v", user.Email, tf)
			tfs = append(tfs, &UserTraffic{
				UserId:          user.UserId,
				DownloadTraffic: user.DownloadTraffic,
				UploadTraffic:   user.UploadTraffic,
			})
			user.resetTraffic()
		}
	}
	postJson(hc, API_ENDPOINT, &syncReq{UserTraffics: tfs})
	log.Printf("[INFO] Call syncUserTrafficToServer ONLINE USER COUNT: %d", len(tfs))
}
