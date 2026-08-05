<script setup lang="ts">
const visible = defineModel<boolean>('visible', { required: true })

const steps = [
  { icon: 'pi-sort-numeric-down', title: 'Priority order', description: 'Enabled provisions are checked top to bottom, per the list order below.' },
  { icon: 'pi-bolt', title: 'Trigger match', description: 'Trigger events / trigger requests are compared to the incoming CWMP event codes / request type. An empty list matches anything.' },
  { icon: 'pi-filter', title: 'Conditions', description: 'Every condition (parameter/operator/value) is checked against the device’s values, all ANDed together.' },
  { icon: 'pi-code', title: 'Scripts run', description: 'The matched provision’s scripts queue in order, then the next provision is evaluated.', highlight: true },
]
</script>

<template>
  <div v-if="visible" class="how-it-works">
    <div class="steps">
      <template v-for="(step, index) in steps" :key="step.title">
        <div class="step" :class="{ highlight: step.highlight }">
          <i class="pi step-icon" :class="step.icon"></i>
          <div class="step-title">{{ step.title }}</div>
          <div class="step-description">{{ step.description }}</div>
        </div>
        <i v-if="index < steps.length - 1" class="pi pi-angle-right connector"></i>
      </template>
    </div>
    <div class="footer-note">
      <i class="pi pi-ban"></i>
      <span>Disabled provisions are skipped entirely and never evaluated.</span>
    </div>
  </div>
</template>

<style scoped>
.how-it-works {
  background: var(--p-content-background, #fff);
  border: 1px solid var(--p-surface-200, #e5e7eb);
  border-radius: 12px;
  padding: 1rem;
  margin-bottom: 1rem;
}

.steps {
  display: flex;
  align-items: stretch;
  gap: 0.5rem;
}

.step {
  flex: 1;
  background: var(--p-surface-50, #f8f9fb);
  border-radius: 9px;
  padding: 0.75rem;
  text-align: center;
}

.step.highlight {
  background: var(--p-primary-50, #eef2ff);
  color: var(--p-primary-700, #4338ca);
}

.step-icon {
  font-size: 1.1rem;
  color: var(--p-primary-color, #6366f1);
  margin-bottom: 0.35rem;
  display: block;
}

.step.highlight .step-icon {
  color: var(--p-primary-700, #4338ca);
}

.step-title {
  font-size: 0.8rem;
  font-weight: 600;
  margin-bottom: 0.15rem;
}

.step-description {
  font-size: 0.7rem;
  color: var(--p-text-muted-color, #6b7280);
  line-height: 1.35;
}

.connector {
  align-self: center;
  color: var(--p-text-muted-color, #9ca3af);
  font-size: 0.9rem;
}

.footer-note {
  margin-top: 0.75rem;
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.75rem;
  color: var(--p-text-muted-color, #9ca3af);
}
</style>
