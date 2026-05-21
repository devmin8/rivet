import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { getCurrentUser, signIn, signOut } from '~/features/auth/api'
import { authKeys } from '~/lib/query-keys'

export function useCurrentUser() {
  return useQuery({
    queryKey: authKeys.currentUser,
    queryFn: getCurrentUser,
    retry: false,
    staleTime: 30_000,
  })
}

export function useSignIn() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: signIn,
    onSuccess: (user) => {
      queryClient.setQueryData(authKeys.currentUser, user)
    },
  })
}

export function useSignOut() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: signOut,
    onSettled: () => {
      queryClient.clear()
    },
  })
}
