import { createApp } from 'vue';
import { createPinia } from 'pinia';
import App from './App.vue';
import './style.css';
import './assets/prism-theme.css';
import { platform } from './utils/platform';

document.documentElement.classList.add(`platform-${platform}`);

createApp(App).use(createPinia()).mount('#app');
