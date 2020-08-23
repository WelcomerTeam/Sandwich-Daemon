// Sandwich-Daemon script.js by TehRockettek
// https://github.com/TheRockettek/Sandwich-Daemon

// Install any plugins
Vue.use(VueChartJs);

Vue.component("line-chart", {
    extends: VueChartJs.Line,
    mixins: [VueChartJs.mixins.reactiveData],
    props: ['data', 'options'],
    mounted() {
        this.renderChart(this.data, {
            scales: {
                xAxes: [{
                    type: "time",
                }]
            }
        })
    },
})

Vue.component("card-display", {
    props: ['title', 'value', 'bg'],
    template: `
        <div class="col justify-content-center d-flex">
            <div :class="bg+' card text-white m-1'" style="width: 18rem;">
                <div class="card-header">{{ title }} </div>
                <div class="card-body">
                    <h5 class="card-title">{{ value }}</h5>
                </div>
            </div>
        </div>`,
})

vue = new Vue({
    el: '#app',
    data() {
        return {
            loading: true,
            error: false,
            data: {},
            analytics: {
                chart: {},
                uptime: "...",
                visible: "...",
                events: "...",
                online: "...",
                colour: "bg-success",
            },
            loadingAnalytics: true,

            statusShard: ["Idle", "Waiting", "Connecting", "Connected", "Ready", "Reconnecting", "Closed"],
            statusGroup: ["Idle", "Starting", "Connecting", "Ready", "Replaced", "Closing", "Closed"],
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
                .then(result => {
                    this.analytics = result.data.response;

                    let up = 0
                    let total = 0
                    let guilds = 0
                    this.analytics.colour = "bg-success";
                    for (let index in this.analytics.clusters) {
                        cluster = this.analytics.clusters[index]
                        guilds += cluster.guilds
                        for (let shard in cluster.status) {
                            if (cluster.status[shard] >= 5) {
                                this.analytics.colour = "bg-alert";
                            } else {
                                up++
                            }
                            total++
                        }
                    }
                    this.analytics.visible = guilds
                    this.analytics.online = up + "/" + total

                    this.error = this.error | !result.data.success;
                })
                .catch(error => console.log(error))
                .finally(() => this.loadingAnalytics = false)
        }
    }
})
