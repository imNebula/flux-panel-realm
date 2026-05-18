# flux-panel-realm 转发面板

本仓库是 [bqlpfy/flux-panel](https://github.com/bqlpfy/flux-panel) 的衍生版本。感谢原作者的开源贡献。  
原项目基于 Gost 实现，本项目已将其核心组件完全迁移为基于 [zhboner/realm](https://github.com/zhboner/realm) v2 的转发面板，提供更高的端口转发性能并降低内存占用。

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

## 快速安装

> 节点端命令建议通过面板页面快速生成。

**面板端**（需要 Docker 和 Docker Compose，服务器需能访问 GitHub 和 GHCR）：

```bash
curl -fL https://raw.githubusercontent.com/imNebula/flux-panel-realm/refs/heads/main/panel_install.sh -o panel_install.sh && chmod +x panel_install.sh && ./panel_install.sh
```

**节点端**：

```bash
curl -fL https://raw.githubusercontent.com/imNebula/flux-panel-realm/refs/heads/main/install.sh -o install.sh && chmod +x install.sh && ./install.sh install --server-addr 面板地址:端口 --secret 节点密钥
```

### 卸载面板

```bash
curl -fL https://raw.githubusercontent.com/imNebula/flux-panel-realm/refs/heads/main/panel_uninstall.sh -o panel_uninstall.sh && chmod +x panel_uninstall.sh && ./panel_uninstall.sh
```

如需同时删除数据库、日志卷、镜像和本地配置文件：

```bash
./panel_uninstall.sh --all
```

### 默认管理员账号

- **账号**: `admin_user`
- **密码**: `admin_user`

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
