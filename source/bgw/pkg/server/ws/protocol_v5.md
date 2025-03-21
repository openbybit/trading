## URL

### **格式**

wss://{host}/v5/private

### **域名**

- stream.bybit.com
- stream.bytick.com
- stream.bybitgloble.com

### **示例**

wss://stream.bybit.com/v5/private

## 通信协议

### 编码格式

请求和返回均使用json格式编码

### 请求参数

| Parameter | Required | Type    | **Description**  |
| --------- | -------- | ------- | ---------------- |
| req_id    | N        | string  | 唯一UUID          |
| op        | Y        | string  | 指令类型          |
| args      | N        | list    | 指令对应的参数     |

### 返回参数

| Parameter | Type     | **Description**        |
| --------- | -------  | ---------------------  |
| req_id    | string   | 唯一UUID,同传入的UUID    |
| op        | string   | 指令类型                 |
| args      | list     | 指令对应的返回参数        |
| ret_msg   | string   | 用于返回错误信息          |
| success   | bool     | 用户标识成功或失败        |
| conn_id   | string   | 连接唯一ID              |

### 指令说明

| ReqOP      | RspType    | **Description** |
| ---------- | ---------- | --------------- |
| ping       | pong       | 心跳             |
| subscribe  | subscribe  | 订阅             |
| unsubscribe| unsubscribe| 取消订阅         |
| login      | login      | token认证,web页面使用 |
| auth       | auth       | ApiKey认证,做市商使用 |

### **ping**

请求

```json
{
    "req_id": "{{uuid}}", 
    "op": "ping"
}
```

返回

```json
{
    "req_id": "{{uuid}}",
    "op": "pong",
    "args": ["1658391478723"]，
    "conn_id": "{{conn_id}}"
}
```

### **login**

请求

```json
{
    "req_id": "{{uuid}}", 
    "op": "login",
    "args": [
        "{{token}}"
    ]
}
```

返回

```json
{
    "req_id": "{{uuid}}", 
    "op": "login",
    "ret_msg": "",
    "success": true,
    "conn_id": "{{conn_id}}"
}
```

### **auth**

请求

```json
{
    "req_id": "{{uuid}}", 
    "op": "auth",
    "args": [
        "{{apiKey}}", //apiKey
        1535975085152, //expires
        "{{signature}}" //signature
    ]
}
```

返回

```json
{
    "req_id": "{{uuid}}", 
    "op": "auth",
    "ret_msg": "",
    "success": true,
    "conn_id": "{{conn_id}}"
}
```

### **subscribe**

请求

```json
{
    "req_id": "{{uuid}}", 
    "op": "subscribe",
    "args": [
        "topic1",
        "topic2",
        "topic3"   
    ]
}
```

返回

```json
{
    "req_id": "{{uuid}}", 
    "ret_msg": "",
    "op": "subscribe",
    "success": true,
    "conn_id": "{{conn_id}}"
}
```

### **unsubscribe**

请求

```json
{
    "req_id": "{{uuid}}", 
    "op": "unsubscribe",
    "args": [
        "topic1",
        "topic2",
        "topic3"
    ]
}
```

返回

```json
{
    "req_id": "{{uuid}}", 
    "op": "unsubscribe",
    "ret_msg": "",
    "success": true,
    "conn_id": "{{conn_id}}"
}
```
