<template>
  <div class="auth-shell">
    <div class="auth-card">
      <RouterView />
    </div>
  </div>
</template>

<style scoped>
.auth-shell {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--p-surface-100, #f1f5f9);
}

.auth-card {
  width: 100%;
  max-width: 380px;
  padding: 2rem;
  border-radius: 12px;
  background: var(--p-surface-0, #fff);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
}
</style>
