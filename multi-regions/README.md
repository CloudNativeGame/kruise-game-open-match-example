# 多集群/多地域匹配

本例适用于多集群场景，

- 前端模拟器(frontend)模拟玩家产生匹配请求，随机生成ticket包含cluster-name属性，60%概率为Host集群；40%概率为region-b集群
- match-function将两个同地域属性的ticket组成一个match
- director将对应集群中的可用的游戏服分配给对应的match，并标记其为Allocated

## 部署

### 部署游戏服集合

在多个集群中部署同名称的GameServerSet，示例Yaml如 ./deploy/game.yaml所示（不同集群环境下Yaml参数或有不同）

### 赋予Host集群访问Slave集群权限

将从集群的kubeconfig以secret部署在Host集群的open-match命名空间中
```yaml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: region-b # replace with your cluster name
  namespace: open-match
type: Opaque
data:
  config: YXBpVmVyc2lvbjoxxxxxx... # replace with the baes64 of your own kubernetes kubeconfig.
EOF
```

### 部署frontend / match function / director 示例

```bash
# 注意参数替换
kubectl apply -f ./deploy 
```

## 效果

观察到不同集群中的同名GameServer的OpsState变成Allocated，直到全部GameServer都变为Allocated，director将报错没有充足的后端游戏服可供分配。

