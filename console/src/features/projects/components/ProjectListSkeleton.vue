<template>
  <Table class="min-w-[1080px] table-fixed" aria-label="Loading projects">
    <colgroup>
      <col class="w-[18%]" />
      <col class="w-[8%]" />
      <col class="w-[15%]" />
      <col class="w-[6%]" />
      <col class="w-[9%]" />
      <col class="w-[7%]" />
      <col class="w-[12%]" />
      <col class="w-[12%]" />
      <col class="w-[13%]" />
    </colgroup>
    <TableHeader class="bg-muted/20 dark:bg-muted/10">
      <TableRow class="hover:bg-transparent">
        <TableHead v-for="index in 9" :key="index" class="h-10 px-6 py-2.5">
          <div class="h-3 rounded bg-muted" />
        </TableHead>
      </TableRow>
    </TableHeader>
    <TableBody>
      <TableRow v-for="index in 3" :key="index">
        <TableCell class="px-6 py-4"><div class="h-4 w-40 rounded bg-muted" /></TableCell>
        <TableCell class="py-4"><div class="h-5 w-20 rounded-full bg-muted" /></TableCell>
        <TableCell class="py-4"><div class="h-4 w-36 rounded bg-muted" /></TableCell>
        <TableCell class="py-4"><div class="h-4 w-12 rounded bg-muted" /></TableCell>
        <TableCell class="py-4"><div class="h-4 w-16 rounded bg-muted" /></TableCell>
        <TableCell class="py-4"><div class="h-4 w-10 rounded bg-muted" /></TableCell>
        <TableCell class="py-4"><div class="h-4 w-28 rounded bg-muted" /></TableCell>
        <TableCell class="py-4"><div class="h-4 w-24 rounded bg-muted" /></TableCell>
        <TableCell class="px-6 py-4">
          <div class="flex justify-end gap-2">
            <div class="size-8 rounded bg-muted" />
            <div class="h-8 w-20 rounded bg-muted" />
          </div>
        </TableCell>
      </TableRow>
    </TableBody>
  </Table>
</template>
