<script lang="ts" setup>
import { ref, onMounted, onBeforeMount } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

interface Node {
  space_id: string
  node_id: string
  node_type: "app" | "client"
  docker_pid: number
  app_id: string
  service: string
  domain: string
}

const nodes = ref<Node[]>([])


async function refreshList() {
  try {
    const response = await axios.get('/api/space/list')
    nodes.value = response.data
  } catch (error) {
    console.error('Failed to fetch nodes:', error)
  }
}

async function removeApp(nodeid: string, appid?: string) {
  console.log({
    nodeid: nodeid,
    appid: appid,
  })
  let item = nodes.value.find(v => v.node_id === nodeid)

  let msg = ``
  console.log("kick item", item)
  if (item.node_type == "app") {
    msg = `Are you sure you want to kick app ${appid} & all nodes of app? `
  } else {
    msg = `Are you sure you want to kick client ${nodeid}?`
  }

  try {
    await ElMessageBox.confirm(
      msg,
      'Confirm Kick',
      {
        confirmButtonText: 'Kick',
        cancelButtonText: 'Cancel',
        type: 'warning',
      }
    )
  } catch (_) {
    console.log(" User cancelled the kickiiing action.")
    ElMessage.info('Kickiiiiiiing action cancelled')
    return
  }

  const kickNode = async (nodeid: string, appid?: string) => {
    try {
      let url = `/api/app/remove?nodeid=${nodeid}`
      if (appid) {
        url += `&appid=${appid}`
      }
      await axios.post(url)
      console.log('App removed successfully')
      // Optionally, refresh the list of nodes
      await refreshList()
      ElMessage.success(`${appid} App removed successfully`)
    } catch (err) {
      console.error(err)
      ElMessage.error('Failed to remove app')
    }
  }

  // 只踢client,也就是client
  if (!appid) {
    kickNode(nodeid)
    return
  }

  // 需要将相同appid的节点都踢掉
  nodes.value.forEach(v => {
    kickNode(v.node_id, v.app_id)
  })
}
const configDialog = ref<boolean>(false)
// space的配置
const configYml = ref<string>("")

async function getConfig() {
  configYml.value = (await axios.get("/api/space/config")).data
}

onBeforeMount(() => {
  getConfig()
})

onMounted(async () => {
  await refreshList()
})

</script>

<template>
  <el-card class="usage-example" shadow="hover">
    <template #header>
      <div class="card-header">
        <span>README</span>
      </div>
    </template>
    <div>
      <el-row>
        <el-col :span="12">
          <p>1. 通过Add App添加应用到space中</p>
          <p>2. 根据你的系统，下载对应的客户端程序 lzcspacenode </p>
          <p>3. 运行 lzcspacenode -config config.yml</p>
        </el-col>
        <el-col :span="12">
          将下面配置保存为 config.yml 文件，并使用 -config ./config.yml
          <el-input type="textarea" :rows="4" v-model="configYml" readonly></el-input>
        </el-col>
      </el-row>
    </div>
    <div class="download-buttons">
      <el-link href="/static/spacenode-client-linux" target="_blank">Linux</el-link>&emsp;
      <el-link href="/static/spacenode-client-win.exe" target="_blank">Windows</el-link>
    </div>
  </el-card>
  <div class="space-list">
    <el-table :data="nodes" style="width: 100%">
      <el-table-column label="NodeID" width="180">
        <template #default="{ row }">
          {{ row.node.node_id }}
        </template>
      </el-table-column>
      <el-table-column label="NodeType" width="180">
        <template #default="{ row }">
          {{ row.node.node_type }}
        </template>
      </el-table-column>
      <el-table-column label="NodeInfo">
        <template #default="{ row }">
          <div>
            appid: {{ row.node.app_id }}
          </div>
          <div>
            service: {{ row.node.service }}
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="ip" label="IP" />
      <el-table-column label="Operation">
        <template #default="{ row }">
          <!-- {{ row.node }} -->
          <el-button type="danger" size="small" @click="removeApp(row.node.node_id, row.node.app_id)">Kick</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
<style scoped>
.space-list {
  padding: 20px;
}
</style>
