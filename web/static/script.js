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

Vue.component("cluster-list", {
    props: ['clusters'],
    template: `
    <div>
        <div v-for="cluster in clusters">
            <div class="accordion my-4" :id="'cluster-' + cluster.configuration.identifier">
                <div class="card">
                    <div class="card-header" :id="'header-cluster-' + cluster.configuration.identifier">
                        <h2 class="mb-0">
                            <button class="btn btn-link btn-block text-left text-decoration-none text-dark"
                                type="button" data-toggle="collapse"
                                :data-target="'#collapse-cluster-' + cluster.configuration.identifier"
                                aria-expanded="true"
                                :aria-controls="'collapse-cluster-' + cluster.configuration.identifier">
                                <span v-if="cluster.status == 0"
                                    class="badge bg-dark">{{ statusShard[cluster.status] }}</span>
                                <span v-else-if="cluster.status == 1"
                                    class="badge bg-info">{{ statusShard[cluster.status] }}</span>
                                <span v-else-if="cluster.status == 2"
                                    class="badge bg-info">{{ statusShard[cluster.status] }}</span>
                                <span v-else-if="cluster.status == 4"
                                    class="badge bg-warn">{{ statusShard[cluster.status] }}</span>
                                <span v-else-if="cluster.status == 5"
                                    class="badge bg-warn">{{ statusShard[cluster.status] }}</span>
                                <span v-else-if="cluster.status == 6"
                                    class="badge bg-dark">{{ statusShard[cluster.status] }}</span>
                                <span v-else-if="cluster.status == 7"
                                    class="badge bg-danger">{{ statusShard[cluster.status] }}</span>
                                <span v-else class="badge bg-success">{{ statusShard[cluster.status] }}</span>
                                <span>{{ cluster.configuration.display_name }}</span>
                            </button>
                        </h2>
                    </div>

                    <div :id="'collapse-cluster-' + cluster.configuration.identifier" class="collapse"
                        :aria-labelledby="'header-cluster-' + cluster.configuration.identifier"
                        :data-parent="'#cluster-' + cluster.configuration.identifier">
                        <div class="card-body">
                            <ul class="nav nav-tabs" id="pills-tab" role="tablist">
                                <li class="nav-item" role="presentation">
                                    <a class="nav-link active" :id="'cluster-' + cluster.configuration.identifier + 'pills-status-tab'" data-toggle="pill" :href="'#cluster-' + cluster.configuration.identifier + 'pills-status'" role="tab"
                                        :aria-controls="'cluster-' + cluster.configuration.identifier + 'pills-status'" aria-selected="true">Status</a>
                                </li>
                                <li class="nav-item" role="presentation">
                                    <a class="nav-link" :id="'cluster-' + cluster.configuration.identifier + 'pills-settings-tab bg-dark'" data-toggle="pill" :href="'#cluster-' + cluster.configuration.identifier + 'pills-settings'"
                                        role="tab" :aria-controls="'cluster-' + cluster.configuration.identifier + 'pills-settings'" aria-selected="false">Settings</a>
                                </li>
                            </ul>
                            <div class="tab-content">
                                <div class="tab-pane fade p-4 active show" :id="'cluster-' + cluster.configuration.identifier + 'pills-status'" role="tabpanel" :aria-labelledby="'cluster-' + cluster.configuration.identifier + 'pills-status-tab'">
                                    <div class="row row-cols-1 row-cols-sm-2 row-cols-md-3 row-cols-lg-4 g-4 justify-content-center">
                                        <card-display :title="'Shard Groups'" :value="Object.keys(cluster.shardgroups).length" :bg="'bg-dark'"></card-display>
                                        <card-display :title="'Average Latency'" :value="Object.values(cluster.shardgroups).reduce((a, shardgroup) => Object.values(shardgroup.shards).reduce((a,shard) => a + (new Date(shard.last_heartbeat_ack) - new Date(shard.last_heartbeat_sent)), a), 0) + ' ms'" :bg="'bg-dark'"></card-display>
                                    </div>


                                </div>
                                <div class="tab-pane fade p-4" :id="'cluster-' + cluster.configuration.identifier + 'pills-settings'" role="tabpanel" :aria-labelledby="'cluster-' + cluster.configuration.identifier + 'pills-settings-tab'">
                                    settings
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
    `,
    data() {
        return {
            status: -1,
            shardgroups: {},
            statusShard: ["Idle", "Waiting", "Connecting", "Connected", "Ready", "Reconnecting", "Closed", "Error"],
            statusGroup: ["Idle", "Starting", "Connecting", "Ready", "Replaced", "Closing", "Closed"],
        }
    },
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
            statusGroup: ["Idle", "Starting", "Connecting", "Ready", "Replaced", "Closing", "Closed"],
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
        }
    }
})
