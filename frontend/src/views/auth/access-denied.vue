<template>
  <div class="denied-wrapper">
    <div class="denied-box">
      <div class="denied-code">403</div>
      <div class="denied-sep"></div>
      <div class="denied-msg">Access Denied</div>
    </div>
  </div>
</template>

<script>
export default { name: 'AccessDenied' }
</script>

<style scoped>
.denied-wrapper {
  background: #000;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
}
.denied-box { text-align: center; user-select: none; }
.denied-code {
  font-size: 6rem;
  font-weight: 900;
  color: #1a1a1a;
  font-family: monospace;
  letter-spacing: -4px;
}
.denied-sep {
  width: 60px;
  height: 2px;
  background: #111;
  margin: 1.2rem auto;
}
.denied-msg {
  font-size: 0.75rem;
  color: #222;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  font-family: monospace;
}
</style>
