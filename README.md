reads stdin and sends emails, [wechat-notify](https://github.com/caiguanhao/wechat-notify) compatible

```
go build -ldflags "-X main.accessKeyId=xxxxxxxxxxxxxxxx
-X main.accessKeySecret=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
-X main.fromEmailAddress=no-reply@example.com
-X main.fromEmailAlias=No-Reply"
```
