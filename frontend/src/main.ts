import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './tailwind.css'
import App from './App.vue'
import { router } from './router'
import { Lineicons } from '@lineiconshq/vue-lineicons'

createApp(App).use(createPinia()).use(router).component('Lineicons', Lineicons).mount('#app')
