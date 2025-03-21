
说明文档：
https://c1ey4wdv9g.larksuite.com/docx/RenDd7oS7ofmpsxAkiQuqxmxsNe

demo：

    import (
        "code.bydev.io/fbu/gateway/gway.git/gcompliance"
    )
    
    var wall gcompliance.Wall
    
    func GetKycWall() gcompliance.Wall {
        once.Do(func() {
            wall = gcompliance.NewKycWall("user_service", "default_group", "public")
            wall.SetDiscovery(discoveryFunc)
            wall.SetGeoIP(geoIPFunc)
    
           // 注册kafa回调 增量
           go func() {
               if err := kafka.Consume(context.Background(), brokers, topic, nil, handleWhiteMsg); err != nil {
                  alert.Alert(context.Background(), "member unified Consume error", err, nil)
               }
           }()
        })

        return wall
    }

    func handleWhiteMsg(msg *sarama.ConsumerMessage) error {
        return wall.HandleUserWhiteListEvent(msg.Value)
    }


    func Check(ctx context.Context) (res gcompliance.Result, hit bool, err error) {
        // get broker id
        brokerID := getBrokerID(ctx)
    
        // get uid
        uid := getUid(ctx)
        
        // get scene
        scene := getScene(ctx)
        
        // get ip
        ip := getIP(ctx)
        
        res, hit, err = gkc.CheckStrategy(ctx, brokerid, scene, uid, ip)
        return         
    }