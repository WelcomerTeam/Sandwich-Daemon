// Sandwich-Daemon script.js by TehRockettek
// https://github.com/TheRockettek/Sandwich-Daemon

// Install any plugins
Vue.use(VueChartJs);

Vue.component("line-chart", {
    extends: VueChartJs.Line,
    mixins: [VueChartJs.mixins.reactiveData],
    props: ['data', 'options'],
    mounted() {
        this.renderChart(this.data, this.options)
    },
})

// {
//     labels: [1, 2, 3, 4, 5, 6, 7, 8, 9,],
//     datasets: [
//         {
//             label: 'Data One',
//             backgroundColor: '#435325',
//             data: [getRandomInt(), getRandomInt(), getRandomInt(), getRandomInt(),],
//         }
//     ]
// }, { responsive: true }

vue = new Vue({
    el: '#app',
    data() {
        return {
            loading: true,
            error: false,
            data: {},
            analytics: {},
            loadingAnalytics: true,
        }
    },
    mounted() {
        this.fetchConfiguration();
        this.fetchAnalytics();
    },
    methods: {
        fetchConfiguration() {
            axios
                .get('/api/configuration')
                .then(result => { this.data = result.data.response; this.error = !result.data.success })
                .catch(error => console.log(error))
                .finally(() => this.loading = false)
        },
        fetchAnalytics() {
            axios
                .get('/api/analytics')
                .then(result => { this.analytics = result.data.response; this.error = this.error | !result.data.success })
                .catch(error => console.log(error))
                .finally(() => this.loadingAnalytics = false)
        }
    }
})

function getRandomInt() {
    return Math.floor(Math.random() * 100)
}