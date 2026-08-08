<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { EditorState } from '@codemirror/state'
import { EditorView, keymap, lineNumbers } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { StreamLanguage } from '@codemirror/language'
import { lua } from '@codemirror/legacy-modes/mode/lua'
import { oneDark } from '@codemirror/theme-one-dark'

const luaLanguage = StreamLanguage.define(lua)

const value = defineModel<string>({ required: true })
const props = defineProps<{ readonly?: boolean }>()
const container = ref<HTMLDivElement>()
let view: EditorView | undefined

onMounted(() => {
  if (!container.value) return

  view = new EditorView({
    parent: container.value,
    state: EditorState.create({
      doc: value.value,
      extensions: [
        lineNumbers(),
        history(),
        keymap.of([...defaultKeymap, ...historyKeymap]),
        luaLanguage,
        oneDark,
        EditorState.readOnly.of(!!props.readonly),
        EditorView.editable.of(!props.readonly),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) value.value = update.state.doc.toString()
        }),
      ],
    }),
  })
})

watch(value, (newValue) => {
  if (!view) return
  const current = view.state.doc.toString()
  if (newValue !== current) {
    view.dispatch({ changes: { from: 0, to: current.length, insert: newValue } })
  }
})

onBeforeUnmount(() => view?.destroy())
</script>

<template>
  <div ref="container" class="editor"></div>
</template>

<style scoped>
.editor {
  border: 1px solid var(--p-surface-300, #cbd5e1);
  border-radius: 6px;
  overflow: hidden;
  font-size: 0.85rem;
}

.editor :deep(.cm-editor) {
  min-height: 160px;
}
</style>
