<script setup lang="ts">
import Dialog from 'primevue/dialog'
import Tabs from 'primevue/tabs'
import TabList from 'primevue/tablist'
import Tab from 'primevue/tab'
import TabPanels from 'primevue/tabpanels'
import TabPanel from 'primevue/tabpanel'
import JsonViewer from '@/components/common/JsonViewer.vue'
import type { LogEntry } from '@/api/types/log'

defineProps<{ entry: LogEntry | null }>()
const visible = defineModel<boolean>('visible', { required: true })
</script>

<template>
  <Dialog v-model:visible="visible" header="Log details" modal style="width: 42rem">
    <Tabs v-if="entry" value="xml">
      <TabList>
        <Tab value="xml">Raw XML</Tab>
        <Tab value="json">JSON</Tab>
      </TabList>
      <TabPanels>
        <TabPanel value="xml">
          <pre class="xml-block">{{ entry.full_xml }}</pre>
        </TabPanel>
        <TabPanel value="json">
          <JsonViewer :data="entry" />
        </TabPanel>
      </TabPanels>
    </Tabs>
  </Dialog>
</template>

<style scoped>
.xml-block {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: ui-monospace, monospace;
  font-size: 0.8rem;
  max-height: 60vh;
  overflow: auto;
}
</style>
