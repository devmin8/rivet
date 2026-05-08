import '~/styles/main.css'

import { VueQueryPlugin } from '@tanstack/vue-query'
import { createApp } from 'vue'

import { queryClient } from '~/lib/query-client'
import App from '~/App.vue'
import router from '~/routes/router'

const app = createApp(App)

app.use(VueQueryPlugin, { queryClient })
app.use(router)

app.mount('#app')
