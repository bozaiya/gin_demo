<template>
  <div id="app">
    <Poll />
  </div>
</template>

<script>
import PollView from './components/PollView.vue'

export default {
  components: { Poll: PollView }
}
</script>