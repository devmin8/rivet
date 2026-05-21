<script setup lang="ts">
import { useColorMode } from '@vueuse/core'
import {
  Blocks,
  LoaderCircle,
  LogOut,
  Monitor,
  Moon,
  Sun,
} from 'lucide-vue-next'
import { useRouter } from 'vue-router'

import { useSignOut } from '~/features/auth/queries'

const router = useRouter()
const signOut = useSignOut()
const colorMode = useColorMode({ emitAuto: true })

const sidebarItemClass =
  'h-8 gap-2 px-2.5 text-sm font-normal text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground data-[active=true]:bg-sidebar-accent data-[active=true]:font-medium data-[active=true]:text-sidebar-accent-foreground [&>svg]:size-4 [&>svg]:text-sidebar-foreground/55 data-[active=true]:[&>svg]:text-sidebar-accent-foreground'
const themeMenuItemClass = 'gap-2 py-1.5 pr-2 text-[13px] font-normal'

async function handleSignOut() {
  if (signOut.isPending.value) {
    return
  }

  await signOut.mutateAsync().catch(() => undefined)
  await router.replace('/signin')
}
</script>

<template>
  <SidebarProvider>
    <Sidebar collapsible="offcanvas">
      <SidebarHeader class="h-14 justify-center border-b border-sidebar-border px-3">
        <ConsoleLogo href="/" />
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Overview</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  as-child
                  :is-active="$route.name === 'projects'"
                  :class="sidebarItemClass"
                >
                  <RouterLink to="/">
                    <Blocks aria-hidden="true" />
                    <span>Projects</span>
                  </RouterLink>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter class="border-t border-sidebar-border">
        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <SidebarMenuButton :class="sidebarItemClass">
                  <Monitor v-if="colorMode === 'auto'" aria-hidden="true" />
                  <Moon v-else-if="colorMode === 'dark'" aria-hidden="true" />
                  <Sun v-else aria-hidden="true" />
                  <span>Theme</span>
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                side="right"
                align="start"
                :side-offset="8"
                class="w-36 rounded-lg p-1"
              >
                <DropdownMenuRadioGroup v-model="colorMode">
                  <DropdownMenuRadioItem
                    value="light"
                    :class="themeMenuItemClass"
                  >
                    <Sun class="size-3.5" aria-hidden="true" />
                    Light
                  </DropdownMenuRadioItem>
                  <DropdownMenuRadioItem
                    value="dark"
                    :class="themeMenuItemClass"
                  >
                    <Moon class="size-3.5" aria-hidden="true" />
                    Dark
                  </DropdownMenuRadioItem>
                  <DropdownMenuRadioItem
                    value="auto"
                    :class="themeMenuItemClass"
                  >
                    <Monitor class="size-3.5" aria-hidden="true" />
                    System
                  </DropdownMenuRadioItem>
                </DropdownMenuRadioGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
          <SidebarMenuItem>
            <SidebarMenuButton
              type="button"
              :class="sidebarItemClass"
              :disabled="signOut.isPending.value"
              @click="handleSignOut"
            >
              <LoaderCircle
                v-if="signOut.isPending.value"
                class="animate-spin"
                aria-hidden="true"
              />
              <LogOut v-else aria-hidden="true" />
              <span>Sign out</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>

      <SidebarRail />
    </Sidebar>

    <SidebarInset class="min-h-dvh">
      <SidebarTrigger
        class="fixed left-3 top-3 z-30 bg-background shadow-sm md:hidden"
      />
      <RouterView />
    </SidebarInset>
  </SidebarProvider>
</template>
