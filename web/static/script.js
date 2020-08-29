// Sandwich-Daemon script.js by TehRockettek
// https://github.com/TheRockettek/Sandwich-Daemon

// Install any plugins
Vue.use(VueChartJs);

Vue.component("line-chart", {
    extends: VueChartJs.Line,
    mixins: [VueChartJs.mixins.reactiveProp],
    props: ['chartData', 'options'],
    mounted() {
        this.renderChart(this.chartData, this.options)
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
    </div>
    `,
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

            statusShard: ["Idle", "Waiting", "Connecting", "Connected", "Ready", "Reconnecting", "Closed", "Error"],
            colourShard: ["dark", "info", "info", "success", "success", "warn", "dark", "danger"],

            statusGroup: ["Idle", "Starting", "Connecting", "Ready", "Replaced", "Closing", "Closed"],

            colourCluster: ["dark", "info", "info", "success", "warn", "warn", "dark", "danger"],
        }
    },
    filters: {
        pretty: function (value) {
            return JSON.stringify(value, null, 2);
        }
    },
    mounted() {
        this.fetchConfiguration();
        this.fetchAnalytics();
        this.$nextTick(function () {
            window.setInterval(() => {
                this.fetchAnalytics();
            }, 15000);
        })
    },
    methods: {
        sendRPC(method, params, id) {
            axios
                .post('/api/rpc', {
                    'method': method,
                    'params': params,
                    'id': id,
                })
                .then(result => {
                    return result
                })
                .catch(err => console.log(error))
        },
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
                        shardgroups = Object.values(cluster.status)

                        if (shardgroups.length > 0) {
                            for (let shard in shardgroups) {
                                if (1 < shard.status < 6) {
                                    up++
                                }
                                total++
                            }
                        } else {
                            total++
                        }
                    }
                    this.analytics.visible = guilds
                    this.analytics.online = up + "/" + total

                    this.error = this.error | !result.data.success;
                })
                .catch(error => console.log(error))
                .finally(() => this.loadingAnalytics = false)
        },
        fromClusters(clusters) {
            _clusters = {}
            Object.entries(clusters).forEach((item) => {
                key = item[0]
                value = item[1]

                shardgroups = Object.values(value.shard_groups)
                if (shardgroups.length == 0) {
                    status = 0
                } else {
                    status = shardgroups.slice(-1)[0].status
                }

                _clusters[key] = {
                    configuration: value.configuration,
                    shardgroups: value.shard_groups,
                    status: status,
                }
            })
            return _clusters
        },
        calculateAverage(cluster) {
            totalShards = 0;
            totalLatency = 0;

            shardgroups = Object.values(cluster.shardgroups)
            for (sgindex in shardgroups) {
                shardgroup = shardgroups[sgindex]
                if (shardgroup.status < 6) {
                    shards = Object.values(shardgroup.shards)
                    for (shindex in shards) {
                        shard = shards[shindex]
                        totalLatency = totalLatency + (new Date(shard.last_heartbeat_ack) - new Date(shard.last_heartbeat_sent))
                        totalShards = totalShards + 1
                    }
                }
            }
            return (totalLatency / totalShards) || '-'
        }
    }
})
