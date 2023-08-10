# 1v1 匹配

本例适用于1v1匹配场景，其中：

- 前端模拟器(frontend)模拟玩家产生匹配请求，生成tickets
- match-function将两个ticket组成一个match
- director将集群中的可用的游戏服分配给对应的match，并标记其为Allocated

## 部署

需要针对集群环境修改GameServerSet的Network参数，默认示例使用的是AlibabaCloud-SLB网络模型。详细配置可参考[OKG网络功能相关文档](https://openkruise.io/zh/kruisegame/user-manuals/network)

将该示例部署至 ~/.kube/config 所指Kubernetes集群

```bash
kubectl create ns open-match-demo

kubectl apply -f ./deploy
```

## 效果

观察到集群中的GameServer的OpsState变成Allocated，直到全部GameServer都变为Allocated，director将报错没有充足的后端游戏服可供分配。

