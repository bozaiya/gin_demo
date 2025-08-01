<template>
  <div class="poll">
    <h2>{{ question }}</h2>
    <div v-for="option in options" :key="option.id" class="option">
      <input
          type="radio"
          :id="'option' + option.id"
          :value="option.id"
          v-model="selectedOption"
          :disabled="hasVoted"
      />
      <label :for="'option' + option.id">{{ option.text }}</label>
    </div>
    <button @click="submitVote" :disabled="!selectedOption || hasVoted">提交投票</button>
    <div v-if="hasVoted" class="voted-message">您已投票，谢谢参与！</div>
    <div ref="chart" class="chart"></div>
  </div>
</template>

<script>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import axios from 'axios'
import * as echarts from 'echarts'

export default {
  setup() {
    const question = ref('您最喜欢哪个选项？')
    const options = ref([])
    const selectedOption = ref(null)
    const hasVoted = ref(false)
    let chart = null
    let ws = null

    // 初始化图表
    const initChart = () => {
      // 销毁旧图表实例
      if (chart) chart.dispose()
      // 初始化新实例
      chart = echarts.init(document.querySelector('.chart'))
      // 更新数据
      updateChart() 
    }

    // 更新图表数据
    const updateChart = () => {
      const option = {
        // X 轴显示选项文本
        xAxis: {
          type: 'category',
          data: options.value.map(opt => opt.text)
        },
        // Y 轴显示票数
        yAxis: { type: 'value' },
        series: [{ type: 'bar', data: options.value.map(opt => opt.votes) }]
      }
      // 应用配置
      chart.setOption(option)
    }

    // 获取初始数据
    const fetchPollData = async () => {
      try {
        const res = await axios.get('http://localhost:8080/api/poll')
        options.value = res.data.options
        // 初始化或更新图表
        initChart()
      } catch (error) {
        console.error('获取数据失败:', error)
      }
    }

    // 提交投票
    const submitVote = async () => {
      try {
        await axios.post('http://localhost:8080/api/poll/vote', { optionId: selectedOption.value })
        hasVoted.value = true
        // 本地存储标记
        localStorage.setItem('hasVoted', 'true')  
      } catch (error) {
        console.error('投票失败:', error)
      }
    }

    // 生命周期
    onMounted(() => {
      hasVoted.value = localStorage.getItem('hasVoted') === 'true'  // 恢复投票状态
      fetchPollData()

      // 建立 WebSocket 连接
      ws = new WebSocket('ws://localhost:8080/ws/poll')
      ws.onmessage = (event) => {
        const data = JSON.parse(event.data)
        // 更新数据
        options.value = data.options
        // 刷新图表
        updateChart()     
      }
    })

    // 清理资源
    onBeforeUnmount(() => {
      // 关闭 WebSocket
      if (ws) ws.close()
      // 销毁图表
      if (chart) chart.dispose()   
    })

    return { question, options, selectedOption, hasVoted, submitVote }
  }
}
</script>

<style>
.chart {
  width: 600px;
  height: 400px;
  margin-top: 20px;
}
.option {
  margin: 10px 0;
}
</style>