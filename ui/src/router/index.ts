import { createRouter, createWebHistory } from 'vue-router'
import SpaceList from '@/views/SpaceList.vue'
import SpaceAdd from '@/views/SpaceAdd.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/space/list',
      name: 'SpaceList',
      component: SpaceList
    },
    {
      path: '/space/add',
      name: 'SpaceAdd',
      component: SpaceAdd
    },
    {
      path: '/',
      redirect: '/space/list'
    }
  ]
})

export default router
