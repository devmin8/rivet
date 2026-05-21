import { createRouter, createWebHistory } from 'vue-router'

import { getCurrentUser } from '~/features/auth/api'
import { ApiError } from '~/lib/errors'
import { queryClient } from '~/lib/query-client'
import { authKeys } from '~/lib/query-keys'

const ConsoleLayout = () => import('~/layouts/console-layout.vue')
const Dashboard = () => import('~/routes/dashboard.vue')
const SignIn = () => import('~/routes/sign-in.vue')

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: ConsoleLayout,
      meta: {
        requiresAuth: true,
      },
      children: [
        {
          path: '',
          name: 'projects',
          component: Dashboard,
        },
      ],
    },
    {
      path: '/signin',
      name: 'signin',
      component: SignIn,
      meta: {
        guestOnly: true,
      },
    },
  ],
})

router.beforeEach(async (to) => {
  // public routes
  if (!to.meta.requiresAuth && !to.meta.guestOnly) {
    return true
  }

  try {
    const user = await queryClient.fetchQuery({
      queryKey: authKeys.currentUser,
      queryFn: getCurrentUser,
      staleTime: 30_000,
      retry: false,
    })

    queryClient.setQueryData(authKeys.currentUser, user)

    if (to.meta.guestOnly) {
      return { path: '/' }
    }

    return true
  } catch (error) {
    queryClient.removeQueries({ queryKey: authKeys.currentUser })

    if (to.meta.guestOnly) {
      return true
    }

    if (error instanceof ApiError && error.status === 401) {
      return { path: '/signin' }
    }

    return { path: '/signin' }
  }
})

export default router
