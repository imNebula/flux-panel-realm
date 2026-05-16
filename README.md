# flux-panel-realm 转发面板
> ⚠️ 项目暂无完成开发，请勿使用

本仓库是 [bqlpfy/flux-panel](https://github.com/bqlpfy/flux-panel) 的衍生版本。感谢原作者的开源贡献。  
原项目基于 Gost 实现，本项目已将其核心组件完全迁移为基于 [zhboner/realm](https://github.com/zhboner/realm) v2 的转发面板。底层服务迁移至原生的 Realm 实现。

> ⚠️ 由于 Realm 支持的隧道加密协议较少，部分 Gost 协议无法使用。

---

## 特性

- **核心迁移**：底层迁移至 Realm，提供更高的端口转发性能并降低内存占用。
- **流量与延迟监控**：支持端口精准流量统计。
- **资源限制**：支持按 **隧道账号级别** 管理流量转发数量，可用于用户/隧道配额控制。
- **协议支持**：支持 **TCP** 和 **UDP** 协议的转发。
- **多种模式**：支持两种转发模式：**端口转发** 与 **隧道转发**。
- **流量计费**：支持配置 **单向或双向流量计费方式**，灵活适配不同计费模型。

---

## 部署流程

### Docker Compose部署

#### 快速部署

> 节点段建议使用面板快速生成

面板端(稳定版，拉取 GitHub 预构建镜像，不在服务器编译)：
```bash
curl -fL https://raw.githubusercontent.com/imNebula/flux-panel-realm/refs/heads/main/panel_install.sh -o panel_install.sh && chmod +x panel_install.sh && ./panel_install.sh
```
节点端(稳定版)：
```bash
curl -fL https://raw.githubusercontent.com/imNebula/flux-panel-realm/refs/heads/main/install.sh -o install.sh && chmod +x install.sh && ./install.sh install --server-addr 面板地址:端口 --secret 节点密钥
```

面板安装只需要 Docker / Docker Compose，并且服务器需要能访问 GitHub Release 和 GHCR 镜像仓库。每次推送到 `main` 都会自动编译面板镜像和节点端 Agent：如果 `.github/workflows/docker-build.yml` 中的 `VERSION` 尚未发布过，则生成稳定版；如果该版本号已经存在，则自动生成 `VERSION-dev.<run_number>.<attempt>` 开发/测试版。

如需固定安装某个 release/tag，可在运行脚本时指定版本：
```bash
VERSION=0.0.1-realm ./panel_install.sh
```

如需安装开发版：
```bash
CHANNEL=development ./panel_install.sh
```
或指定开发版 release：
```bash
VERSION=0.0.1-realm-dev.1 ./panel_install.sh
```

节点端如需安装最新开发版 Agent：
```bash
AGENT_CHANNEL=development ./install.sh install --server-addr 面板地址:端口 --secret 节点密钥
```

如需使用自定义镜像仓库或标签：
```bash
PANEL_IMAGE_REGISTRY=your.registry/flux-panel-realm PANEL_IMAGE_TAG=0.0.1-realm ./panel_install.sh
```

#### 默认管理员账号

- **账号**: admin_user
- **密码**: admin_user

> ⚠️ 首次登录后请立即修改默认密码！

---

## 免责声明

本项目仅供个人学习与研究使用，基于开源项目进行二次开发。  
使用本项目所带来的任何风险均由使用者自行承担，包括但不限于：  

- 配置不当或使用错误导致的服务异常或不可用；  
- 使用本项目引发的网络攻击、封禁、滥用等行为；  
- 服务器因使用本项目被入侵、渗透、滥用导致的数据泄露、资源消耗或损失；  
- 因违反当地法律法规所产生的任何法律责任。  

本项目为开源的流量转发工具，仅限合法、合规用途。  
使用者必须确保其使用行为符合所在国家或地区的法律法规。  

**作者不对因使用本项目导致的任何法律责任、经济损失或其他后果承担责任。**  
**禁止将本项目用于任何违法或未经授权的行为，包括但不限于网络攻击、数据窃取、非法访问等。**  

如不同意上述条款，请立即停止使用本项目。  
作者对因使用本项目所造成的任何直接或间接损失概不负责，亦不提供任何形式的担保、承诺或技术支持。  
请务必在合法、合规、安全的前提下使用本项目。
