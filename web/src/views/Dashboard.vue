<template>
  <div class="container-xl p-3">
    <div v-if="$store.state.userLoading">
      <div class="spinner-border mx-auto mt-5 d-flex" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>
    <div v-else-if="!$store.state.userAuthenticated">
      <div class="text-center my-5">
        <svg-icon
          class="mb-2 text-muted"
          type="mdi"
          :width="64"
          :height="64"
          :path="mdiAlertCircle"
        />
        <h3>You are not authenticated</h3>
        <p class="m-auto my-2 col-10 text-muted">
          It seems you are not where you want to be...
        </p>
      </div>
    </div>
    <div v-else-if="!this.loading">
      <!-- Create ShardGroup Dialogue -->
      <div
        class="modal fade"
        id="createShardGroupDialogue"
        tabindex="-1"
        aria-labelledby="createShardGroupDialogueLabel"
        aria-hidden="true"
      >
        <div class="modal-dialog">
          <div class="modal-content">
            <div class="modal-header">
              <h5 class="modal-title" id="createShardGroupDialogueLabel">
                New ShardGroup for {{ createShardGroupDialogueData.manager }}
              </h5>
              <button
                type="button"
                class="close"
                data-dismiss="modal"
                aria-label="Close"
              >
                <span aria-hidden="true">&times;</span>
              </button>
            </div>
            <div class="modal-body">
              <div class="mb-3">
                <label class="col-sm-12 form-label">Shard Count</label>
                <input
                  class="form-check-input"
                  type="checkbox"
                  v-model="createShardGroupDialogueData.autoShard"
                />
                <label class="form-check-label mb-2">Auto Determine</label>
                <input
                  type="number"
                  class="form-control"
                  v-model="createShardGroupDialogueData.shardCount"
                  :disabled="createShardGroupDialogueData.autoShard"
                />
              </div>
              <div class="mb-3">
                <label class="col-sm-12 form-label">Shard IDs</label>
                <input
                  class="form-check-input"
                  type="checkbox"
                  v-model="createShardGroupDialogueData.autoIDs"
                />
                <label class="form-check-label mb-2">Auto Determine</label>
                <input
                  type="text"
                  class="form-control"
                  v-model="createShardGroupDialogueData.shardIDs"
                  :disabled="createShardGroupDialogueData.autoIDs"
                  placeholder="from-to,from-to,from-to"
                />
              </div>
              <!-- <div class="form-check mt-5">
                              <input class="form-check-input" type="checkbox"
                                  v-model="createShardGroupDialogueData.startImmediately">
                              <label class="form-check-label">Start ShardGroup Immediately</label>
                          </div> -->
            </div>
            <div class="modal-footer">
              <button
                type="button"
                class="btn btn-secondary"
                data-dismiss="modal"
              >
                Close
              </button>
              <button
                type="button"
                class="btn btn-success"
                v-on:click="createShardGroup()"
              >
                Create ShardGroup
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Stop Shardgroup Dialogue -->
      <div
        class="modal fade"
        id="stopShardGroupDialogue"
        tabindex="-1"
        aria-labelledby="stopShardGroupDialogueLabel"
        aria-hidden="true"
      >
        <div class="modal-dialog">
          <div class="modal-content">
            <div class="modal-header">
              <h5 class="modal-title" id="stopShardGroupDialogueLabel">
                Stop ShardGroup {{ stopShardGroupDialogueData.shardgroup }} for
                {{ stopShardGroupDialogueData.manager }}
              </h5>
              <button
                type="button"
                class="close"
                data-dismiss="modal"
                aria-label="Close"
              >
                <span aria-hidden="true">&times;</span>
              </button>
            </div>
            <div class="modal-body">
              <span>Are you sure you want to kill this shard group?</span>
            </div>
            <div class="modal-footer">
              <button
                type="button"
                class="btn btn-secondary"
                data-dismiss="modal"
              >
                Close
              </button>
              <button
                type="button"
                class="btn btn-danger"
                v-on:click="stopShardGroup()"
              >
                Stop ShardGroup
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Create Manager Dialogue -->
      <div
        class="modal fade"
        id="createManagerDialogue"
        tabindex="-1"
        aria-labelledby="createManagerDialogueLabel"
        aria-hidden="true"
      >
        <div class="modal-dialog">
          <div class="modal-content">
            <div class="modal-header">
              <h5 class="modal-title" id="createManagerDialogueLabel">
                New Manager
              </h5>
              <button
                type="button"
                class="close"
                data-dismiss="modal"
                aria-label="Close"
              >
                <span aria-hidden="true">&times;</span>
              </button>
            </div>
            <div class="modal-body">
              <form>
                <div class="mb-3">
                  <label class="col-sm-12 form-label"
                    >Identifier <span class="text-danger">*</span></label
                  >
                  <input
                    type="text"
                    class="form-control"
                    v-model="createManagerDialogueData.identifier"
                    placeholder="Enter unique identifiable name for manager"
                  />
                </div>
                <div class="mb-3">
                  <label class="col-sm-12 form-label"
                    >Token <span class="text-danger">*</span></label
                  >
                  <input
                    class="form-control"
                    type="password"
                    v-model="createManagerDialogueData.token"
                    placeholder="Enter bot token"
                    autocomplete="false"
                  />
                </div>
                <div class="mb-3">
                  <label class="col-sm-12 form-label">Redis Prefix</label>
                  <input
                    class="form-control"
                    type="text"
                    v-model="createManagerDialogueData.prefix"
                    :placeholder="'eg: ' + createManagerDialogueData.identifier"
                  />
                </div>
                <div class="mb-3">
                  <label class="col-sm-12 form-label"
                    >NATs Client Name <span class="text-danger">*</span></label
                  >
                  <input
                    class="form-control"
                    type="text"
                    v-model="createManagerDialogueData.client"
                    :placeholder="'eg: ' + createManagerDialogueData.identifier"
                  />
                </div>
                <div class="mb-3">
                  <label class="col-sm-12 form-label">NATs Channel Name</label>
                  <input
                    class="form-control"
                    type="text"
                    v-model="createManagerDialogueData.channel"
                    :placeholder="'Defaults to ' + configuration.nats.channel"
                  />
                </div>

                <input
                  class="form-check-input"
                  type="checkbox"
                  v-model="createManagerDialogueData.persist"
                />
                <label class="form-check-label mb-2"
                  >Persist <span class="text-danger">*</span></label
                >
                <p class="text-muted">
                  When enabled, changes to Manager will be saved to the
                  configuration. Un-check if you do not want to save the
                  manager.
                </p>
              </form>
              <p class="text-danger">* Required</p>
            </div>
            <div class="modal-footer">
              <button
                type="button"
                class="btn btn-secondary"
                data-dismiss="modal"
              >
                Close
              </button>
              <button
                type="button"
                class="btn btn-success"
                v-on:click="createManager()"
              >
                Create Manager
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Delete Manager Dialogue -->
      <div
        class="modal fade"
        id="deleteManagerDialogue"
        tabindex="-1"
        aria-labelledby="deleteManagerDialogueLabel"
        aria-hidden="true"
      >
        <div class="modal-dialog">
          <div class="modal-content">
            <div class="modal-header">
              <h5 class="modal-title" id="deleteManagerDialogueLabel">
                Delete Manager {{ deleteManagerDialogueData.manager }}?
              </h5>
              <button
                type="button"
                class="close"
                data-dismiss="modal"
                aria-label="Close"
              >
                <span aria-hidden="true">&times;</span>
              </button>
            </div>
            <div class="modal-body">
              <label class="form-check-label mb-2"
                >Confirm deleting manager by typing
                <b>{{ deleteManagerDialogueData.manager }}</b></label
              >
              <input
                type="text"
                class="form-control"
                v-model="deleteManagerDialogueData.confirm"
                :placeholder="
                  'Type: \'' + deleteManagerDialogueData.manager + '\''
                "
              />
            </div>
            <div class="modal-footer">
              <button
                type="button"
                class="btn btn-secondary"
                data-dismiss="modal"
              >
                Close
              </button>
              <button
                type="button"
                class="btn btn-danger"
                v-on:click="deleteManager()"
              >
                Delete Manager
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Restart Manager Dialogue -->
      <div
        class="modal fade"
        id="restartManagerDialogue"
        tabindex="-1"
        aria-labelledby="restartManagerDialogueLabel"
        aria-hidden="true"
      >
        <div class="modal-dialog">
          <div class="modal-content">
            <div class="modal-header">
              <h5 class="modal-title" id="restartManagerDialogueLabel">
                Restart Manager {{ restartManagerDialogueData.manager }}?
              </h5>
              <button
                type="button"
                class="close"
                data-dismiss="modal"
                aria-label="Close"
              >
                <span aria-hidden="true">&times;</span>
              </button>
            </div>
            <div class="modal-body">
              <label class="form-check-label mb-2"
                >Confirm restarting manager by typing its name. This will stop
                any currently running shardgroups</label
              >
              <input
                type="text"
                class="form-control"
                v-model="restartManagerDialogueData.confirm"
                :placeholder="
                  'Type: \'' + restartManagerDialogueData.manager + '\''
                "
              />
            </div>
            <div class="modal-footer">
              <button
                type="button"
                class="btn btn-secondary"
                data-dismiss="modal"
              >
                Close
              </button>
              <button
                type="button"
                class="btn btn-info"
                v-on:click="restartManager()"
              >
                Restart Manager
              </button>
            </div>
          </div>
        </div>
      </div>

      <ul class="nav nav-pills mb-3" id="pills-tab" role="tablist">
        <li class="nav-item" role="presentation">
          <a
            class="nav-link active"
            id="pills-home-tab"
            data-toggle="pill"
            href="#pills-home"
            role="tab"
            aria-controls="pills-home"
            aria-selected="true"
            >Analytics</a
          >
        </li>
        <li class="nav-item" role="presentation">
          <a
            class="nav-link"
            id="pills-managers-tab"
            data-toggle="pill"
            href="#pills-managers"
            role="tab"
            aria-controls="pills-managers"
            aria-selected="false"
            >Managers</a
          >
        </li>
        <li class="nav-item" role="presentation">
          <a
            class="nav-link"
            id="pills-settings-tab bg-dark"
            data-toggle="pill"
            href="#pills-settings"
            role="tab"
            aria-controls="pills-settings"
            aria-selected="false"
            >Daemon Settings</a
          >
        </li>
      </ul>
      <div class="tab-content" id="pills-tabContent">
        <div
          class="tab-pane fade show active"
          id="pills-home"
          role="tabpanel"
          aria-labelledby="pills-home-tab"
        >
          <div class="m-5">
            <h3 class="text-center text-dark">Sandwich Daemon</h3>
            <div
              class="row row-cols-1 row-cols-sm-2 row-cols-md-3 row-cols-lg-4 g-4 justify-content-center"
            >
              <card-display
                :title="'Guilds'"
                :value="analytics.guilds"
                bg="bg-dark"
              />
              <card-display
                :title="'Online'"
                :value="analytics.online"
                :bg="analytics.colour"
              />
              <card-display
                :title="'Uptime'"
                :value="analytics.uptime"
                bg="bg-dark"
              />
              <card-display
                :title="'Events Processed'"
                :value="analytics.events"
                bg="bg-dark"
              />
            </div>
            <div>
              <line-chart
                v-if="analytics.chart"
                :chart-data="analytics.chart"
                :height="150"
                :options="chartOptions.events"
              >
              </line-chart>
            </div>
          </div>

          <div v-if="rest_tunnel_enabled" class="m-5">
            <div>
              <h3 class="text-center text-dark">RestTunnel</h3>
              <div
                class="row row-cols-1 row-cols-sm-2 row-cols-md-3 row-cols-lg-4 g-4 justify-content-center"
              >
                <card-display
                  :title="'Requests Processed'"
                  :value="resttunnel.numbers.requests"
                  :bg="'bg-dark'"
                >
                </card-display>
                <card-display
                  :title="'Hits'"
                  :value="resttunnel.numbers.hits"
                  :bg="'bg-dark'"
                >
                </card-display>
                <card-display
                  :title="'Misses'"
                  :value="resttunnel.numbers.misses"
                  :bg="'bg-dark'"
                >
                </card-display>
                <card-display
                  :title="'Uptime'"
                  :value="resttunnel.uptime"
                  :bg="'bg-dark'"
                >
                </card-display>
                <card-display
                  :title="'Waiting'"
                  :value="resttunnel.numbers.waiting"
                  :bg="'bg-dark'"
                >
                </card-display>
              </div>
              <div
                class="row row-cols-2 row-cols-sm-1 row-cols-md-2 row-cols-xl-3 g-4 justify-content-center"
              >
                <div class="col justify-content-center d-flex">
                  <line-chart
                    v-if="resttunnel.charts.hits"
                    :chart-data="resttunnel.charts.hits"
                    :height="300"
                    :options="chartOptions.ratelimitHits"
                  >
                  </line-chart>
                </div>

                <div class="col justify-content-center d-flex">
                  <line-chart
                    v-if="resttunnel.charts.misses"
                    :chart-data="resttunnel.charts.misses"
                    :height="300"
                    :options="chartOptions.ratelimitMisses"
                  >
                  </line-chart>
                </div>

                <div class="col justify-content-center d-flex">
                  <line-chart
                    v-if="resttunnel.charts.waiting"
                    :chart-data="resttunnel.charts.waiting"
                    :height="300"
                    :options="chartOptions.waitingRequests"
                  >
                  </line-chart>
                </div>

                <div class="col justify-content-center d-flex">
                  <line-chart
                    v-if="resttunnel.charts.requests"
                    :chart-data="resttunnel.charts.requests"
                    :height="300"
                    :options="chartOptions.totalRequests"
                  >
                  </line-chart>
                </div>

                <div class="col justify-content-center d-flex">
                  <line-chart
                    v-if="resttunnel.charts.callbacks"
                    :chart-data="resttunnel.charts.callbacks"
                    :height="300"
                    :options="chartOptions.callbackBuffer"
                  >
                  </line-chart>
                </div>

                <div class="col justify-content-center d-flex">
                  <line-chart
                    v-if="resttunnel.charts.average_response"
                    :chart-data="resttunnel.charts.average_response"
                    :height="300"
                    :options="chartOptions.averageResponse"
                  >
                  </line-chart>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div
          class="tab-pane fade"
          id="pills-managers"
          role="tabpanel"
          aria-labelledby="pills-managers-tab"
        >
          <button
            type="button"
            class="btn btn-dark"
            v-on:click="createManagerDialogue()"
          >
            Create Manager
          </button>
          <div v-for="(manager, index) in managers" v-bind:key="index">
            <div
              class="accordion my-4"
              :id="'manager-' + manager.configuration.identifier"
            >
              <div class="card">
                <div
                  class="card-header"
                  :id="'header-manager-' + manager.configuration.identifier"
                >
                  <h2 class="mb-0">
                    <button
                      class="btn btn-link btn-block text-left text-decoration-none text-dark"
                      type="button"
                      data-toggle="collapse"
                      :data-target="
                        '#collapse-manager-' + manager.configuration.identifier
                      "
                      aria-expanded="true"
                      :aria-controls="
                        'collapse-manager-' + manager.configuration.identifier
                      "
                    >
                      <span
                        :class="
                          'badge mr-2 bg-' + colourCluster[manager.status] ||
                            'default'
                        "
                        >{{ statusGroup[manager.status] }}</span
                      >
                      <span>{{ manager.configuration.display_name }}</span>
                    </button>
                  </h2>
                </div>
                <div
                  :id="'collapse-manager-' + manager.configuration.identifier"
                  class="collapse"
                  :aria-labelledby="
                    'header-manager-' + manager.configuration.identifier
                  "
                  :data-parent="'#manager-' + manager.configuration.identifier"
                >
                  <div class="card-body">
                    <ul class="nav nav-tabs" id="pills-tab" role="tablist">
                      <li class="nav-item" role="presentation">
                        <a
                          class="nav-link active"
                          :id="
                            'manager-' +
                              manager.configuration.identifier +
                              '-pills-status-tab'
                          "
                          data-toggle="pill"
                          :href="
                            '#manager-' +
                              manager.configuration.identifier +
                              '-pills-status'
                          "
                          role="tab"
                          :aria-controls="
                            'manager-' +
                              manager.configuration.identifier +
                              '-pills-status'
                          "
                          aria-selected="true"
                          >Status</a
                        >
                      </li>
                      <li class="nav-item" role="presentation">
                        <a
                          class="nav-link"
                          :id="
                            'manager-' +
                              manager.configuration.identifier +
                              '-pills-settings-tab'
                          "
                          data-toggle="pill"
                          :href="
                            '#manager-' +
                              manager.configuration.identifier +
                              '-pills-settings'
                          "
                          role="tab"
                          :aria-controls="
                            'manager-' +
                              manager.configuration.identifier +
                              '-pills-settings'
                          "
                          aria-selected="false"
                          >Settings</a
                        >
                      </li>
                    </ul>
                    <div class="tab-content">
                      <div
                        class="tab-pane fade p-4 active show"
                        :id="
                          'manager-' +
                            manager.configuration.identifier +
                            '-pills-status'
                        "
                        role="tabpanel"
                        :aria-labelledby="
                          'manager-' +
                            manager.configuration.identifier +
                            '-pills-status-tab'
                        "
                      >
                        <ul class="list-group mb-4">
                          <li
                            class="list-group-item d-flex justify-content-between align-items-center"
                          >
                            Shard Groups
                            <span class="badge bg-dark rounded-pill">{{
                              Object.keys(manager.shard_groups).length
                            }}</span>
                          </li>
                          <li
                            class="list-group-item d-flex justify-content-between align-items-center"
                          >
                            Average Latency
                            <span class="badge bg-dark rounded-pill"
                              >{{ calculateAverage(manager) }} ms</span
                            >
                          </li>
                          <li
                            class="list-group-item d-flex justify-content-between align-items-center"
                          >
                            Sessions
                            <span class="badge bg-dark rounded-pill"
                              >{{
                                manager.gateway.session_start_limit.remaining
                              }}/{{
                                manager.gateway.session_start_limit.total
                              }}
                              ({{
                                manager.gateway.session_start_limit
                                  .max_concurrency
                              }})</span
                            >
                          </li>
                        </ul>

                        <button
                          type="button"
                          class="btn btn-dark mr-1"
                          v-on:click="
                            createShardGroupDialogue(
                              manager.configuration.identifier
                            )
                          "
                        >
                          Scale Manager
                        </button>
                        <button
                          type="button"
                          class="btn btn-info mr-1"
                          v-on:click="
                            refreshGateway(manager.configuration.identifier)
                          "
                        >
                          Refresh Gateway
                        </button>
                        <button
                          type="button"
                          class="btn btn-dark mr-1"
                          v-on:click="
                            deleteManagerDialogue(
                              manager.configuration.identifier
                            )
                          "
                        >
                          Delete Manager
                        </button>
                        <button
                          type="button"
                          class="btn btn-dark mr-1"
                          v-on:click="
                            restartManagerDialogue(
                              manager.configuration.identifier
                            )
                          "
                        >
                          Restart Manager
                        </button>

                        <div v-if="manager.error != ''">
                          <div class="p-3 mb-2 bg-danger text-white my-4">
                            {{ manager.error }}
                          </div>
                        </div>

                        <div
                          v-for="(shardgroup, index) in manager.shard_groups"
                          v-bind:key="index"
                          class="border border-bottom bg-light rounded-lg p-3 my-4"
                        >
                          <div class="mb-3">
                            <h5>
                              <span
                                :class="
                                  'badge rounded-pill bg-' +
                                    colourGroup[shardgroup.status] || 'default'
                                "
                                >{{ statusGroup[shardgroup.status] }}</span
                              >
                              ShardGroup {{ shardgroup.id }}
                            </h5>
                            <div v-if="shardgroup.status != 6">
                              <button
                                type="button"
                                class="btn btn-dark"
                                v-on:click="
                                  stopShardGroupDialogue(
                                    manager.configuration.identifier,
                                    shardgroup.id
                                  )
                                "
                              >
                                Stop ShardGroup
                              </button>
                            </div>
                            <div v-if="shardgroup.status >= 6">
                              <button
                                type="button"
                                class="btn btn-dark"
                                v-on:click="
                                  deleteShardGroup(
                                    manager.configuration.identifier,
                                    shardgroup.id
                                  )
                                "
                              >
                                Delete ShardGroup
                              </button>
                            </div>
                          </div>

                          <div v-if="shardgroup.status != 6">
                            <ul class="list-group mb-4">
                              <li
                                class="list-group-item d-flex justify-content-between align-items-center"
                              >
                                Status
                                <status-graph
                                  :value="shardgroup"
                                  :colours="colourShard"
                                  style="width: 50%"
                                >
                                </status-graph>
                              </li>
                              <li
                                class="list-group-item d-flex justify-content-between align-items-center"
                              >
                                Uptime
                                <span class="badge bg-dark rounded-pill">{{
                                  since(shardgroup.uptime)
                                }}</span>
                              </li>
                              <li
                                class="list-group-item d-flex justify-content-between align-items-center"
                              >
                                Shards
                                <span class="badge bg-dark rounded-pill"
                                  >{{
                                    Object.keys(shardgroup.shard_ids).length
                                  }}/{{ shardgroup.shard_count }}</span
                                >
                              </li>
                              <li
                                class="list-group-item d-flex justify-content-between align-items-center"
                              >
                                Average Latency
                                <span class="badge bg-dark rounded-pill"
                                  >{{
                                    calculateAverageShardGroup(shardgroup)
                                  }}
                                  ms</span
                                >
                              </li>
                            </ul>

                            <div v-if="shardgroup.error">
                              <div class="p-3 mb-2 bg-danger text-white">
                                {{ shardgroup.error }}
                              </div>
                            </div>

                            <div
                              v-if="
                                shardgroup.shard_count >
                                  manager.gateway.shards * 4
                              "
                            >
                              <div class="p-3 mb-2 bg-info text-white">
                                You are launching this shardgroup with an
                                unnecessarily large number of shards. This may
                                cause the shardgroup to take longer to become
                                ready. It is recommend you use a smaller number
                                closer to the recommended shard count of 1 shard
                                per 1000 guilds which is
                                <b>{{ manager.gateway.shards }}</b
                                >.
                              </div>
                            </div>

                            <table class="table table-borderless">
                              <thead>
                                <tr class="table-dark">
                                  <th scope="col">Shard</th>
                                  <th scope="col">Status</th>
                                  <th scope="col">RTT Latency</th>
                                </tr>
                              </thead>
                              <tbody>
                                <tr
                                  class="table-default"
                                  v-for="(shard, index) in shardgroup.shards"
                                  v-bind:key="index"
                                >
                                  <th scope="row">{{ shard.shard_id }}</th>
                                  <td>
                                    <span
                                      :class="
                                        'badge bg-' + colourShard[shard.status]
                                      "
                                      >{{ statusShard[shard.status] }}</span
                                    >
                                  </td>
                                  <td>
                                    {{
                                      new Date(shard.last_heartbeat_ack) -
                                        new Date(shard.last_heartbeat_sent)
                                    }}ms
                                  </td>
                                </tr>
                              </tbody>
                            </table>
                          </div>
                        </div>
                      </div>
                      <div
                        class="tab-pane fade"
                        :id="
                          'manager-' +
                            manager.configuration.identifier +
                            '-pills-settings'
                        "
                        role="tabpanel"
                        :aria-labelledby="
                          'manager-' +
                            manager.configuration.identifier +
                            '-pills-settings-tab'
                        "
                      >
                        <ul class="nav nav-tabs" id="tabpanel" role="tablist">
                          <li class="nav-item" role="presentation">
                            <a
                              class="nav-link active"
                              id="general-tab"
                              data-toggle="tab"
                              :href="
                                '#manager-' +
                                  manager.configuration.identifier +
                                  '-Settings-general'
                              "
                              role="tab"
                              aria-selected="true"
                              >General</a
                            >
                          </li>
                          <li class="nav-item" role="presentation">
                            <a
                              class="nav-link"
                              id="bot-tab"
                              data-toggle="tab"
                              :href="
                                '#manager-' +
                                  manager.configuration.identifier +
                                  '-Settings-bot'
                              "
                              role="tab"
                              aria-selected="false"
                              >Bot</a
                            >
                          </li>
                          <li class="nav-item" role="presentation">
                            <a
                              class="nav-link"
                              id="caching-tab"
                              data-toggle="tab"
                              :href="
                                '#manager-' +
                                  manager.configuration.identifier +
                                  '-Settings-caching'
                              "
                              role="tab"
                              aria-selected="false"
                              >Caching</a
                            >
                          </li>
                          <li class="nav-item" role="presentation">
                            <a
                              class="nav-link"
                              id="events-tab"
                              data-toggle="tab"
                              :href="
                                '#manager-' +
                                  manager.configuration.identifier +
                                  '-Settings-events'
                              "
                              role="tab"
                              aria-selected="false"
                              >Events</a
                            >
                          </li>
                          <li class="nav-item" role="presentation">
                            <a
                              class="nav-link"
                              id="messaging-tab"
                              data-toggle="tab"
                              :href="
                                '#manager-' +
                                  manager.configuration.identifier +
                                  '-Settings-messaging'
                              "
                              role="tab"
                              aria-selected="false"
                              >Messaging</a
                            >
                          </li>
                          <li class="nav-item" role="presentation">
                            <a
                              class="nav-link"
                              id="sharding-tab"
                              data-toggle="tab"
                              :href="
                                '#manager-' +
                                  manager.configuration.identifier +
                                  '-Settings-sharding'
                              "
                              role="tab"
                              aria-selected="false"
                              >Sharding</a
                            >
                          </li>
                          <li class="nav-item" role="presentation">
                            <a
                              class="nav-link"
                              id="raw-tab"
                              data-toggle="tab"
                              :href="
                                '#manager-' +
                                  manager.configuration.identifier +
                                  '-Settings-raw'
                              "
                              role="tab"
                              aria-selected="false"
                              >RAW (Read Only)</a
                            >
                          </li>
                        </ul>
                        <div class="tab-content p-5" id="pills-managerSettings">
                          <div
                            class="tab-pane fade show active"
                            :id="
                              'manager-' +
                                manager.configuration.identifier +
                                '-Settings-general'
                            "
                            role="tabpanel"
                            aria-labelledby="general-tab"
                          >
                            <!-- General -->
                            <div class="pb-4">
                              <form-input
                                v-model="manager.configuration.auto_start"
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-autostart'
                                "
                                :label="'Auto Start'"
                              />
                              <p class="text-muted">
                                When enabled, the Manager will start up a
                                shardgroup automatically.
                              </p>
                              <form-input
                                v-model="manager.configuration.persist"
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-persist'
                                "
                                :label="'Persist'"
                              />
                              <p class="text-muted">
                                When enabled, changes to Manager will be saved
                                else the configuration will be discarded when
                                the daemon is next started.
                              </p>
                            </div>
                            <form-input
                              v-model="manager.configuration.identifier"
                              :type="'text'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-identifier'
                              "
                              :label="'Identifier'"
                              :disabled="true"
                            />
                            <p class="text-muted">
                              <b
                                >You cannot modify the identifier as that can
                                possibly break many things.</b
                              >
                              The name the manager is internally referenced by.
                              NATs packets will also include this identifier in
                              its messages.
                            </p>
                            <form-input
                              v-model="manager.configuration.display_name"
                              :type="'text'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-display_name'
                              "
                              :label="'Display Name'"
                            />
                            <form-input
                              v-model="manager.configuration.token"
                              :type="'password'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-token'
                              "
                              :label="'Token'"
                            />
                            <form-submit
                              v-on:click="saveClusterSettings(manager)"
                            >
                            </form-submit>
                          </div>
                          <div
                            class="tab-pane fade"
                            :id="
                              'manager-' +
                                manager.configuration.identifier +
                                '-Settings-bot'
                            "
                            role="tabpanel"
                            aria-labelledby="bot-tab"
                          >
                            <!-- Bot -->
                            <div class="pb-4">
                              <form-input
                                v-model="manager.configuration.bot.compression"
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-bot.compression'
                                "
                                :label="'Compression'"
                              />
                              <p class="text-muted">
                                Enables zstd compression on the gateway
                                websocket. Recommended.
                              </p>
                              <form-input
                                v-model="
                                  manager.configuration.bot.guild_subscriptions
                                "
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-bot.guild_subscriptions'
                                "
                                :label="'Guild Subscriptions'"
                              />
                              <p class="text-muted">
                                <b>Not recommended, use intents.</b> Events such
                                as PRESENCE_UPDATE, TYPING, GUILD_MEMBER_JOIN
                                etc. are not sent to the bot.
                              </p>
                            </div>
                            <form-input
                              v-model="manager.configuration.bot.presence"
                              :type="'presence'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-bot.presence'
                              "
                              :label="'Presence'"
                            />
                            <form-input
                              v-model="manager.configuration.bot.intents"
                              :type="'intent'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-bot.intents'
                              "
                              :label="'Intents'"
                            />
                            <form-input
                              v-model="
                                manager.configuration.bot.large_threshold
                              "
                              :type="'number'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-bot.large_threshold'
                              "
                              :label="'Large Threshold'"
                            />
                            <form-input
                              v-model="
                                manager.configuration.bot.max_heartbeat_failures
                              "
                              :type="'number'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-bot.max_heartbeat_failures'
                              "
                              :label="'Max Heartbeat Failures'"
                            />
                            <p class="text-muted">
                              Amount of heartbeat durations until the bot
                              forcibly reconnects. 5 is recommended.
                            </p>
                            <form-input
                              v-model="manager.configuration.bot.retries"
                              :type="'number'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-bot.retries'
                              "
                              :label="'Retries'"
                            />
                            <p class="text-muted">
                              Amount of reconnects to attempt before giving up.
                              Recommended to be more than 1 in the event it
                              fails when starting up a ShardGroup causing all
                              other shards to die.
                            </p>
                            <form-submit
                              v-on:click="saveClusterSettings(manager)"
                            >
                            </form-submit>
                          </div>
                          <div
                            class="tab-pane fade"
                            :id="
                              'manager-' +
                                manager.configuration.identifier +
                                '-Settings-caching'
                            "
                            role="tabpanel"
                            aria-labelledby="caching-tab"
                          >
                            <!-- Caching -->
                            <div class="pb-4">
                              <form-input
                                v-model="
                                  manager.configuration.caching.cache_users
                                "
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-caching.cache_users'
                                "
                                :label="'Cache Users'"
                              />
                              <p class="text-muted">
                                If enabled, users will be cached on redis.
                              </p>
                              <form-input
                                v-model="
                                  manager.configuration.caching.cache_members
                                "
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-caching.cache_members'
                                "
                                :label="'Cache Members'"
                              />
                              <p class="text-muted">
                                If enabled, members will be cached on redis.
                              </p>
                              <form-input
                                v-model="
                                  manager.configuration.caching.request_members
                                "
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-caching.request_members'
                                "
                                :label="'Request Members'"
                              />
                              <p class="text-muted">
                                <b
                                  >Due to the new intent changes, enabling this
                                  on larger bots is not recommended.</b
                                >
                                If enabled, guild members will be requested when
                                lazy loading.
                              </p>
                              <form-input
                                v-model="
                                  manager.configuration.caching.store_mutuals
                                "
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-caching.store_mutuals'
                                "
                                :label="'Store Mutuals'"
                              />
                              <p class="text-muted">
                                If enabled, guild ids a member is on is stored
                                on redis.
                              </p>
                            </div>
                            <form-input
                              v-model="
                                manager.configuration.caching.redis_prefix
                              "
                              :type="'text'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-caching.redis_prefix'
                              "
                              :label="'Redis Prefix'"
                            />
                            <p class="text-muted">
                              String all redis requests will be pre-pended with
                              {PREFIX}:{KEY}. Useful when wanting to separate
                              managers from each other. Having multiple managers
                              with the same key can cause destruction.
                            </p>
                            <form-input
                              v-model="
                                manager.configuration.caching.request_chunk_size
                              "
                              :type="'number'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-caching.request_chunk_size'
                              "
                              :label="'Request Chunk Size'"
                            />
                            <p class="text-muted">
                              Number of guilds to request in a
                              REQUEST_GUILD_MEMBERS request. With the new
                              changes, this should be set to 1. Sending more
                              than 1 when it limits to 1 will not fail but will
                              only sent the members of the first ID specified.
                            </p>
                            <form-submit
                              v-on:click="saveClusterSettings(manager)"
                            >
                            </form-submit>
                          </div>
                          <div
                            class="tab-pane fade"
                            :id="
                              'manager-' +
                                manager.configuration.identifier +
                                '-Settings-events'
                            "
                            role="tabpanel"
                            aria-labelledby="events-tab"
                          >
                            <p class="text-muted">
                              Changes to events will be reflected when a new
                              shardgroup is made
                            </p>
                            <!-- Events -->
                            <div class="pb-4">
                              <form-input
                                v-model="
                                  manager.configuration.events.ignore_bots
                                "
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-events.ignore_bots'
                                "
                                :label="'Ignore Bots'"
                              />
                              <p class="text-muted">
                                When enabled, consumers will not receive
                                MESSAGE_CREATE events if the author is a bot
                              </p>
                              <form-input
                                v-model="
                                  manager.configuration.events.check_prefixes
                                "
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-events.check_prefixes'
                                "
                                :label="'Check Prefixes'"
                              />
                              <p class="text-muted">
                                When enabled, consumers will receive only
                                MESSAGE_CREATE events that start with a defined
                                prefix. The prefix is determined from a HGET to
                                {REDIS_PREFIX}:prefix with the guild id as a
                                key.
                              </p>
                              <form-input
                                v-model="
                                  manager.configuration.events
                                    .allow_mention_prefix
                                "
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-events.allow_mention_prefix'
                                "
                                :label="'Allow Mention Prefix'"
                              />
                            </div>
                            <form-input
                              v-model="
                                manager.configuration.events.fallback_prefix
                              "
                              :type="'text'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-events.fallback_prefix'
                              "
                              :label="'Fallback Prefix'"
                              :placeholder="'No prefix'"
                            />
                            <p class="text-muted">
                              If the daemon is unable to fetch a custom prefix
                              from redis, it will use the fallback prefix and
                              mention prefix (if enabled). If fallback prefix is
                              left empty, it will not allow any message to be
                              used as a command.
                            </p>

                            <form-input
                              v-model="
                                manager.configuration.events.event_blacklist
                              "
                              :type="'list'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-events.event_blacklist'
                              "
                              :label="'Event Blacklist'"
                            />
                            <p class="text-muted">
                              Events in event blacklist are completely ignored
                              by Sandwich Daemon. It is recommended you instead
                              use Intents to stop events as it still has to
                              process the events.
                            </p>
                            <form-input
                              v-model="
                                manager.configuration.events.produce_blacklist
                              "
                              :type="'list'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-events.produce_blacklist'
                              "
                              :label="'Produce Blacklist'"
                            />
                            <p class="text-muted">
                              Events in produce blacklist are processed (cached
                              etc.) however are not relayed to consumers. Useful
                              for reducing events that the consumers
                              unnecessarily need but you still need for cache
                              purposes.
                            </p>
                            <form-submit
                              v-on:click="saveClusterSettings(manager)"
                            >
                            </form-submit>
                          </div>
                          <div
                            class="tab-pane fade"
                            :id="
                              'manager-' +
                                manager.configuration.identifier +
                                '-Settings-messaging'
                            "
                            role="tabpanel"
                            aria-labelledby="messaging-tab"
                          >
                            <!-- Messaging -->
                            <div class="pb-4">
                              <form-input
                                v-model="
                                  manager.configuration.messaging
                                    .use_random_suffix
                                "
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-messaging.use_random_suffix'
                                "
                                :label="'Use Random Suffix'"
                              />
                              <p class="text-muted">
                                When enabled, the client name will have a random
                                suffix added. Useful if you have multiple
                                managers using the same client name.
                              </p>
                            </div>
                            <form-input
                              v-model="
                                manager.configuration.messaging.client_name
                              "
                              :type="'text'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-messaging.client_name'
                              "
                              :label="'Client Name'"
                            />
                            <p class="text-muted">
                              Name of the client the manager uses.
                              <b>Client names must be unique.</b> If you are
                              using multiple managers with the same client name,
                              it is recommended you use separate client names or
                              enable random suffix.
                            </p>
                            <form-input
                              v-model="
                                manager.configuration.messaging.channel_name
                              "
                              :type="'text'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-messaging.channel_name'
                              "
                              :label="'Channel Name'"
                            />
                            <p class="text-muted">
                              Custom definition of the channel the manager will
                              use. If empty, it will use the configuration in
                              daemon.nats.channel which managers should use by
                              default.
                            </p>
                            <form-submit
                              v-on:click="saveClusterSettings(manager)"
                            >
                            </form-submit>
                          </div>
                          <div
                            class="tab-pane fade"
                            :id="
                              'manager-' +
                                manager.configuration.identifier +
                                '-Settings-sharding'
                            "
                            role="tabpanel"
                            aria-labelledby="sharding-tab"
                          >
                            <!-- Sharding -->
                            <div class="pb-4">
                              <form-input
                                v-model="
                                  manager.configuration.sharding.auto_sharded
                                "
                                :type="'checkbox'"
                                :id="
                                  'managerConfig-' +
                                    manager.configuration.identifier +
                                    '-sharding.auto_sharded'
                                "
                                :label="'AutoSharded'"
                              />
                              <p class="text-muted">
                                If enabled, will set the shard count to what is
                                recommended on the gateway
                              </p>
                            </div>
                            <form-input
                              v-model="
                                manager.configuration.sharding.shard_count
                              "
                              :type="'number'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-sharding.shard_count'
                              "
                              :label="'Shard Count'"
                            />
                            <p class="text-muted">
                              Shard Count to launch shard groups with. Be aware
                              this has no security and if it has not supplied
                              enough shards, it will error. It is recommended
                              you use autosharded, enabling autosharded will
                              overwrite this value.
                            </p>
                            <form-input
                              v-model="
                                manager.configuration.sharding.manager_count
                              "
                              :type="'number'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-sharding.manager_count'
                              "
                              :label="'Cluster Count'"
                            ></form-input>
                            <p class="text-muted">
                              <b
                                >Use if you have multiple daemons running. This
                                must be the same number on all daemons and
                                starts from 0.</b
                              >
                              Total number of managers running.
                            </p>
                            <form-input
                              v-model="
                                manager.configuration.sharding.manager_id
                              "
                              :type="'number'"
                              :id="
                                'managerConfig-' +
                                  manager.configuration.identifier +
                                  '-sharding.manager_id'
                              "
                              :label="'Cluster ID'"
                            ></form-input>
                            <p class="text-muted">
                              <b
                                >Use if you have multiple daemons running. This
                                must be a different number on all daemons.</b
                              >
                              Cluster ID of current daemon. With only 1 manager,
                              this ID must be 0 similarly to how shard count and
                              shard ID works.
                            </p>
                            <form-submit
                              v-on:click="saveClusterSettings(manager)"
                            >
                            </form-submit>
                          </div>
                          <div
                            class="tab-pane fade"
                            :id="
                              'manager-' +
                                manager.configuration.identifier +
                                '-Settings-raw'
                            "
                            role="tabpanel"
                            aria-labelledby="raw-tab"
                          >
                            <pre>{{ manager }}</pre>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div
          class="tab-pane fade"
          id="pills-settings"
          role="tabpanel"
          aria-labelledby="pills-settings-tab"
        >
          <form @submit.prevent>
            <ul class="nav nav-tabs" id="tabpanel" role="tablist">
              <li class="nav-item" role="presentation">
                <a
                  class="nav-link active"
                  id="logging-tab"
                  data-toggle="tab"
                  href="#daemonSettings-logging"
                  role="tab"
                  aria-selected="true"
                  >Logging</a
                >
              </li>
              <li class="nav-item" role="presentation">
                <a
                  class="nav-link"
                  id="access-tab"
                  data-toggle="tab"
                  href="#daemonSettings-access"
                  role="tab"
                  aria-selected="false"
                >
                  Access
                </a>
              </li>
              <li class="nav-item" role="presentation">
                <a
                  class="nav-link"
                  id="redis-tab"
                  data-toggle="tab"
                  href="#daemonSettings-redis"
                  role="tab"
                  aria-selected="false"
                  >Redis</a
                >
              </li>
              <li class="nav-item" role="presentation">
                <a
                  class="nav-link"
                  id="nats-tab"
                  data-toggle="tab"
                  href="#daemonSettings-nats"
                  role="tab"
                  aria-selected="false"
                  >NATs</a
                >
              </li>
              <li class="nav-item" role="presentation">
                <a
                  class="nav-link"
                  id="http-tab"
                  data-toggle="tab"
                  href="#daemonSettings-http"
                  role="tab"
                  aria-selected="false"
                  >HTTP</a
                >
              </li>
              <li class="nav-item" role="presentation">
                <a
                  class="nav-link"
                  id="resttunnel-tab"
                  data-toggle="tab"
                  href="#daemonSettings-resttunnel"
                  role="tab"
                  aria-selected="false"
                  ><b>RestTunnel</b></a
                >
              </li>
              <li class="nav-item" role="presentation">
                <a
                  class="nav-link"
                  id="raw-tab"
                  data-toggle="tab"
                  href="#daemonSettings-raw"
                  role="tab"
                  aria-selected="false"
                  >RAW (Read Only)</a
                >
              </li>
            </ul>
            <div class="tab-content p-5" id="pills-daemonSettings">
              <div
                class="tab-pane fade show active"
                id="daemonSettings-logging"
                role="tabpanel"
                aria-labelledby="logging-tab"
              >
                <!-- Logging -->
                <div class="pb-4">
                  <form-input
                    v-model="configuration.logging.console_logging"
                    :type="'checkbox'"
                    :id="'loggingConsoleLogging'"
                    :label="'Console Logging'"
                  />
                  <p class="text-muted">
                    If enabled, logs will be shown in console.
                  </p>
                  <form-input
                    v-model="configuration.logging.file_logging"
                    :type="'checkbox'"
                    :id="'loggingFileLogging'"
                    :label="'File Logging'"
                  />
                  <p class="text-muted">
                    If enabled, logs will be saved to a file.
                  </p>
                  <form-input
                    v-model="configuration.logging.encode_as_json"
                    :type="'checkbox'"
                    :id="'loggingEncodeAsJson'"
                    :label="'Encode as JSON'"
                  />
                  <p class="text-muted">
                    If enabled, text will be encoded as json instead of human
                    readable.
                  </p>
                </div>
                <form-input
                  v-model="configuration.logging.directory"
                  :type="'text'"
                  :id="'loggingDirectory'"
                  :label="'Directory'"
                />
                <form-input
                  v-model="configuration.logging.filename"
                  :type="'text'"
                  :id="'loggingFilename'"
                  :label="'Filename'"
                />
                <form-input
                  v-model="configuration.logging.max_size"
                  :type="'number'"
                  :id="'loggingMaxSize'"
                  :label="'Max Size'"
                />
                <p class="text-muted">
                  <b>Stored in Megabytes.</b> Max size each log can be before a
                  new file is made. 0 is unlimited.
                </p>
                <form-input
                  v-model="configuration.logging.max_backups"
                  :type="'number'"
                  :id="'loggingMaxBackups'"
                  :label="'Max Backups'"
                />
                <p class="text-muted">
                  Max number of log files before old ones are deleted.
                </p>
                <form-input
                  v-model="configuration.logging.max_age"
                  :type="'number'"
                  :id="'loggingMaxAge'"
                  :label="'Max Age'"
                />
                <p class="text-muted">
                  Number of days that a log file can exist before it is removed.
                </p>
                <form-submit v-on:click="saveDaemonSettings()"></form-submit>
              </div>
              <div
                class="tab-pane fade"
                id="daemonSettings-resttunnel"
                role="tabpanel"
                aria-labelledby="resttunnel-tab"
              >
                <!-- RestTunnel -->
                <p class="text-muted">
                  RestTunnel is a second-party application that is used in
                  conjunction with Sandwich-Daemon to allow for centralized rate
                  limit management for multiple systems. The Github repository
                  can be found
                  <a href="https://github.com/TheRockettek/RestTunnel">here</a>.
                </p>
                <p class="text-muted">
                  RestTunnel enabled:
                  <b>{{ rest_tunnel_enabled }}</b>
                </p>

                <div class="pb-4">
                  <form-input
                    v-model="configuration.resttunnel.enabled"
                    :type="'checkbox'"
                    :id="'restTunnelEnabled'"
                    :label="'Enabled'"
                  />
                </div>
                <form-input
                  v-model="configuration.resttunnel.url"
                  :type="'text'"
                  :id="'restTunnelURL'"
                  :placeholder="'http://127.0.0.1:8000'"
                  :label="'URL'"
                />
                <button
                  type="button"
                  class="btn btn-dark mr-1"
                  v-on:click="saveDaemonSettings()"
                >
                  Save Changes
                </button>
                <button
                  type="button"
                  class="btn btn-dark mr-1"
                  v-on:click="verifyRestTunnel()"
                >
                  Verify RestTunnel
                </button>
              </div>
              <div
                class="tab-pane fade"
                id="daemonSettings-access"
                role="tabpanel"
                aria-labelledby="access-tab"
              >
                <!-- Access -->
                <p class="text-muted">
                  Access allows you to manage who has access to the dashboard
                </p>
                <div class="pb-4">
                  <form-input
                    v-model="configuration.http.public"
                    :type="'checkbox'"
                    :id="'httpPublic'"
                    :label="'Public Access'"
                  />
                  <p class="text-muted">
                    If enabled, users will not need elevation to access the
                    internal API. <b>This should never need to be enabled.</b>
                  </p>
                </div>

                <ul class="list-group mt-3 mb-2">
                  <li class="list-group-item list-group-item-dark">
                    Users ({{ configuration.elevated_users.length }})
                  </li>
                  <li
                    class="list-group-item"
                    v-for="(id, index) in configuration.elevated_users"
                    v-bind:key="index"
                  >
                    {{ id }}
                  </li>
                </ul>
                <form-input
                  v-model="userid"
                  :type="'number'"
                  :id="'elevatedUsers'"
                  :label="''"
                  :placeholder="'Discord user ID'"
                />
                <form-submit
                  v-on:click="
                    configuration.elevated_users = configuration.elevated_users.filter(
                      item => item !== userid
                    );
                    userid = undefined;
                  "
                  :label="'Remove User'"
                />
                <form-submit
                  v-on:click="
                    if (Number(userid) != userid) {
                      return;
                    }
                    if (configuration.elevated_users.includes(userid)) {
                      return;
                    }
                    configuration.elevated_users.push(userid);
                    userid = undefined;
                  "
                  :label="'Add User'"
                />
                <p class="text-muted">
                  List of discord user IDs who are able to manage the settings
                  on the dashboard. Only give this to users you trust.
                </p>

                <form-submit v-on:click="saveDaemonSettings()"></form-submit>
              </div>
              <div
                class="tab-pane fade"
                id="daemonSettings-redis"
                role="tabpanel"
                aria-labelledby="redis-tab"
              >
                <!-- Redis -->
                <p class="text-muted">
                  Redis is the driver that handles caching of objects within
                  Sandwich Daemon. This allows for multiple programs to share
                  and interact with the same cache without passing on the
                  objects in messages or using RPC.
                </p>
                <div class="pb-4">
                  <form-input
                    v-model="configuration.redis.unique_clients"
                    :type="'checkbox'"
                    :id="'redisUniqueClients'"
                    :label="'Unique Clients'"
                  />
                </div>
                <p class="text-muted">
                  If enabled, each Manager will have their own redis connection
                  else they will share the same connection.
                </p>
                <form-input
                  v-model="configuration.redis.address"
                  :type="'text'"
                  :id="'redisAddress'"
                  :label="'Address'"
                />
                <form-input
                  v-model="configuration.redis.password"
                  :type="'password'"
                  :id="'redisPassword'"
                  :label="'Password'"
                />
                <form-input
                  v-model="configuration.redis.database"
                  :type="'number'"
                  :id="'redisDB'"
                  :label="'Database'"
                />
                <form-submit v-on:click="saveDaemonSettings()"></form-submit>
              </div>
              <div
                class="tab-pane fade"
                id="daemonSettings-nats"
                role="tabpanel"
                aria-labelledby="nats-tab"
              >
                <!-- NATs -->
                <p class="text-muted">
                  NATs is the driver that allows for consumers to process the
                  messages that are sent by the Daemon. This ensures that the
                  consumers receive their messages and they only receive it
                  once.
                </p>
                <form-input
                  v-model="configuration.nats.address"
                  :type="'text'"
                  :id="'natsAddress'"
                  :label="'Address'"
                />
                <form-input
                  v-model="configuration.nats.channel"
                  :type="'text'"
                  :id="'natsChannel'"
                  :label="'Channel'"
                />
                <form-input
                  v-model="configuration.nats.manager"
                  :type="'text'"
                  :id="'natsCluster'"
                  :label="'Cluster'"
                />
                <form-submit v-on:click="saveDaemonSettings()"></form-submit>
              </div>
              <div
                class="tab-pane fade"
                id="daemonSettings-http"
                role="tabpanel"
                aria-labelledby="http-tab"
              >
                <!-- HTTP -->
                <div class="pb-4">
                  <form-input
                    v-model="configuration.http.enabled"
                    :type="'checkbox'"
                    :id="'httpEnabled'"
                    :label="'Enabled'"
                  />
                  <p class="text-muted">
                    Disabling HTTP will still show the web interface however
                    will shown an error.
                  </p>
                </div>
                <form-input
                  v-model="configuration.http.host"
                  :type="'text'"
                  :id="'httpHost'"
                  :label="'Host'"
                />
                <p class="text-muted">
                  <b
                    >It is recommended you keep this on a local IP such as
                    localhost or 127.0.0.1 or disable public access in the
                    Access tab.</b
                  >
                  Entering 0.0.0.0 or the devices IP will make it accessable to
                  users outside of the local network. Ensure you disable public
                  access in the Access tab.
                </p>

                <form-input
                  v-model="configuration.http.secret"
                  :type="'password'"
                  :id="'httpSecret'"
                  :label="'Session Secret'"
                />
                <p class="text-muted">
                  Secret to use for encrypting user sessions.
                  <b>Changing this will log you out</b>
                </p>
                <form-submit v-on:click="saveDaemonSettings()"></form-submit>
              </div>
              <div
                class="tab-pane fade"
                id="daemonSettings-raw"
                role="tabpanel"
                aria-labelledby="raw-tab"
              >
                <!-- RAW -->
                <pre>{{ configuration }}</pre>
              </div>
            </div>
          </form>
        </div>
      </div>
    </div>
    <div
      aria-live="polite"
      aria-atomic="true"
      style="position: relative; min-height: 200px;"
    >
      <div style="position: fixed; top: 20px; right: 20px;">
        <div
          class="toast"
          id="toast"
          role="alert"
          aria-live="assertive"
          aria-atomic="true"
        >
          <div class="toast-header">
            <strong class="mr-auto">{{ toast.title }} </strong>
            <button
              type="button"
              class="ml-2 mb-1 close"
              data-dismiss="toast"
              aria-label="Close"
            >
              <span aria-hidden="true">&times;</span>
            </button>
          </div>
          <div class="toast-body">
            {{ toast.body }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Stop flashing when loading site */
[v-cloak] {
  display: none;
}

.nav-pills .nav-link {
  color: #343a40;
}
.nav-pills .nav-link.active,
.nav-pills .show > .nav-link {
  color: #fff;
  background-color: #343a40 !important;
}
</style>

<script>
import axios from "axios";
import SvgIcon from "@jamescoyle/vue-icon";
import { mdiAlertCircle } from "@mdi/js";

import { Toast, Modal } from "bootstrap";

import CardDisplay from "@/components/CardDisplay.vue";
import FormInput from "@/components/FormInput.vue";
import FormSubmit from "@/components/FormSubmit.vue";
import LineChart from "@/components/LineChart.vue";
import StatusGraph from "@/components/StatusGraph.vue";

export default {
  components: {
    CardDisplay,
    FormInput,
    FormSubmit,
    LineChart,
    StatusGraph,
    SvgIcon
  },
  name: "Dashboard",
  data() {
    return {
      fetch_task: undefined,
      mdiAlertCircle: mdiAlertCircle,
      version: "...",
      loading: true,
      error: false,
      userid: undefined,

      loadingRestTunnel: true,
      loadingAnalytics: true,

      rest_tunnel_enabled: true,
      managers: [],
      configuration: {},

      toast: {
        title: "",
        body: ""
      },
      analytics: {
        chart: {},
        uptime: "...",
        visible: "...",
        events: "...",
        online: "...",
        colour: "bg-success"
      },

      resttunnel: {
        charts: {
          hits: {},
          misses: {},
          waiting: {},
          requests: {},
          callbacks: {},
          average_response: {}
        },
        numbers: {
          hits: 0,
          misses: 0,
          requests: 0,
          waiting: 0
        },
        uptime: "..."
      },

      createShardGroupDialogueData: {
        manager: "",
        autoShard: true,
        shardCount: 1,
        autoIDs: true,
        shardIDs: "",
        startImmediately: true
      },
      stopShardGroupDialogueData: {
        manager: "",
        shardgroup: 0
      },

      createManagerDialogueData: {
        identifier: "",
        persist: true,
        token: "",
        prefix: "",
        client: "",
        channel: ""
      },
      deleteManagerDialogueData: {
        confirm: "",
        manager: ""
      },
      restartManagerDialogueData: {
        confirm: "",
        manager: ""
      },

      statusShard: [
        "Idle",
        "Waiting",
        "Connecting",
        "Connected",
        "Ready",
        "Reconnecting",
        "Closed"
      ],
      colourShard: [
        "dark",
        "info",
        "info",
        "info",
        "success",
        "warn",
        "secondary"
      ],

      statusGroup: [
        "Idle",
        "Starting",
        "Connecting",
        "Ready",
        "Replaced",
        "Closing",
        "Closed",
        "Error"
      ],
      colourGroup: [
        "dark",
        "info",
        "info",
        "success",
        "info",
        "warn",
        "dark",
        "danger"
      ],

      colourCluster: [
        "dark",
        "info",
        "info",
        "success",
        "warn",
        "warn",
        "dark",
        "danger"
      ],

      chartOptions: {
        ratelimitHits: {
          title: { display: true, text: "Ratelimit Hits" },
          elements: {
            point: { radius: 0 },
            line: { tension: 0.2, borderJoinStyle: "round" }
          },
          scales: {
            yAxes: [
              {
                display: true,
                labelString: "events",
                ticks: { fixedStepSize: 1 }
              }
            ],
            xAxes: [{ type: "time" }]
          },
          animation: { duration: 0 }
        },
        ratelimitMisses: {
          title: { display: true, text: "Ratelimit Misses" },
          elements: {
            point: { radius: 0 },
            line: { tension: 0.2, borderJoinStyle: "round" }
          },
          scales: {
            yAxes: [
              {
                display: true,
                labelString: "events",
                ticks: { fixedStepSize: 1 }
              }
            ],
            xAxes: [{ type: "time" }]
          },
          animation: { duration: 0 }
        },
        waitingRequests: {
          title: { display: true, text: "Waiting requests" },
          elements: {
            point: { radius: 0 },
            line: { tension: 0.2, borderJoinStyle: "round" }
          },
          scales: {
            yAxes: [
              {
                display: true,
                labelString: "events",
                ticks: { fixedStepSize: 1 }
              }
            ],
            xAxes: [{ type: "time" }]
          },
          animation: { duration: 0 }
        },
        totalRequests: {
          title: { display: true, text: "Total requests" },
          elements: {
            point: { radius: 0 },
            line: { tension: 0.2, borderJoinStyle: "round" }
          },
          scales: {
            yAxes: [
              {
                display: true,
                labelString: "events",
                ticks: { fixedStepSize: 1 }
              }
            ],
            xAxes: [{ type: "time" }]
          },
          animation: { duration: 0 }
        },
        callbackBuffer: {
          title: { display: true, text: "Callback buffer" },
          elements: {
            point: { radius: 0 },
            line: { tension: 0.2, borderJoinStyle: "round" }
          },
          scales: {
            yAxes: [
              {
                display: true,
                labelString: "events",
                ticks: { fixedStepSize: 1 }
              }
            ],
            xAxes: [{ type: "time" }]
          },
          animation: { duration: 0 }
        },
        averageResponse: {
          title: { display: true, text: "Average response" },
          elements: {
            point: { radius: 0 },
            line: { tension: 0.2, borderJoinStyle: "round" }
          },
          scales: {
            yAxes: [
              {
                display: true,
                labelString: "events",
                ticks: { beginAtZero: true }
              }
            ],
            xAxes: [{ type: "time" }]
          },
          animation: { duration: 0 },
          tooltips: {
            callbacks: {
              label: function(tooltipItems, data) {
                return (
                  data.datasets[tooltipItems.datasetIndex].label +
                  ": " +
                  tooltipItems.yLabel +
                  "ms"
                );
              }
            }
          }
        },
        events: {
          title: { display: true, text: "Events" },
          elements: {
            point: { radius: 0 },
            line: { tension: 0.2, borderJoinStyle: "round" }
          },
          scales: {
            yAxes: [{ display: true, labelString: "events" }],
            xAxes: [{ type: "time" }]
          },
          animation: { duration: 0 }
        }
      }
    };
  },
  filters: {
    pretty: function(value) {
      return JSON.stringify(value, null, 2);
    }
  },
  mounted() {
    this.toastModal = new Toast(document.getElementById("toast"), {
      delay: 2000
    });

    this.fetchConfiguration();
    this.fetch_task = window.setInterval(() => {
      this.pollData();
    }, 5000);
    this.pollData();
  },
  methods: {
    sendRPC(method, data, id) {
      axios
        .post("/api/rpc", {
          method: method,
          data: data,
          id: id
        })
        .then(result => {
          var err = result.data.error;
          if (!result.data.success) {
            this.showToast("Error executing " + method, err);
          } else {
            this.showToast(method, "Executed successfuly");
          }
          return result;
        })
        .catch(error => {
          console.log(error);
          this.showToast("Exception sending RPC", error);
        });
    },

    showToast(title, body) {
      this.toast.title = title;
      this.toast.body = body;
      this.toastModal.show();
    },

    verifyRestTunnel() {
      this.sendRPC("daemon:verify_resttunnel");
    },

    saveClusterSettings(manager) {
      this.sendRPC("manager:update", manager.configuration);
    },

    saveDaemonSettings() {
      this.sendRPC("daemon:update", this.configuration);
    },

    stopShardGroupDialogue(manager, shardgroup) {
      this.stopShardGroupDialogueModal = new Modal(
        document.getElementById("stopShardGroupDialogue"),
        {}
      );

      this.stopShardGroupDialogueData.manager = manager;
      this.stopShardGroupDialogueData.shardgroup = shardgroup;

      this.stopShardGroupDialogueModal.show();
    },
    stopShardGroup() {
      var config = Object.assign({}, this.stopShardGroupDialogueData);
      this.sendRPC("manager:shardgroup:stop", config);
      setTimeout(() => this.pollData(), 1000);

      this.stopShardGroupDialogueModal.hide();
    },
    deleteShardGroup(manager, shardgroup) {
      var config = {
        manager: manager,
        shardgroup: shardgroup
      };
      this.sendRPC("manager:shardgroup:delete", config);
      setTimeout(() => this.pollData(), 1000);
    },

    refreshGateway(manager) {
      var config = {
        manager: manager
      };
      this.sendRPC("manager:refresh_gateway", config);
      setTimeout(() => this.pollData(), 1000);
    },

    createManagerDialogue() {
      this.createManagerDialogueModal = new Modal(
        document.getElementById("createManagerDialogue"),
        {}
      );
      this.createManagerDialogueData.identifier = "";
      this.createManagerDialogueData.persist = true;
      this.createManagerDialogueData.token = "";
      this.createManagerDialogueData.prefix = "";
      this.createManagerDialogueData.client = "";
      this.createManagerDialogueData.channel = "";

      this.createManagerDialogueModal.show();
    },
    createManager() {
      this.sendRPC("manager:create", this.createManagerDialogueData);
      setTimeout(() => this.fetchConfiguration(), 1000);

      this.createManagerDialogueModal.hide();
    },

    deleteManagerDialogue(manager) {
      this.deleteManagerDialogueModal = new Modal(
        document.getElementById("deleteManagerDialogue"),
        {}
      );
      this.deleteManagerDialogueData = {
        confirm: "",
        manager: manager
      };

      this.deleteManagerDialogueModal.show();
    },
    deleteManager() {
      this.sendRPC("manager:delete", this.deleteManagerDialogueData);
      setTimeout(() => this.fetchConfiguration(), 1000);

      this.deleteManagerDialogueModal.hide();
    },

    restartManagerDialogue(manager) {
      this.restartManagerDialogueModal = new Modal(
        document.getElementById("restartManagerDialogue"),
        {}
      );
      this.restartManagerDialogueData = {
        confirm: "",
        manager: manager
      };

      this.restartManagerDialogueModal.show();
    },
    restartManager() {
      this.sendRPC("manager:restart", this.restartManagerDialogueData);
      setTimeout(() => this.fetchConfiguration(), 1000);

      this.restartManagerDialogueModal.hide();
    },

    createShardGroupDialogue(manager) {
      this.createShardGroupDialogueModal = new Modal(
        document.getElementById("createShardGroupDialogue"),
        {}
      );

      this.createShardGroupDialogueData.manager = manager;
      this.createShardGroupDialogueData.autoShard = true;
      this.createShardGroupDialogueData.shardCount = 1;
      this.createShardGroupDialogueData.autoIDs = true;
      this.createShardGroupDialogueData.shardIDs = "";
      this.createShardGroupDialogueData.startImmediately = true;

      this.createShardGroupDialogueModal.show();
    },
    createShardGroup() {
      var config = Object.assign({}, this.createShardGroupDialogueData);
      config.shardCount = Number(config.shardCount);
      this.sendRPC("manager:shardgroup:create", config);
      setTimeout(() => this.pollData(), 1000);

      this.createShardGroupDialogueModal.hide();
    },

    pollData() {
      axios
        .get("/api/poll")
        .then(result => {
          if (result.status == 403) {
            clearInterval(this.fetch_task);
          }
          if (result.data.success == false) {
            return;
          }
          if (this.error) {
            document.location.reload();
          }

          this.error = !result.data.success;

          this.rest_tunnel_enabled = result.data.data.rest_tunnel_enabled;
          this.uptime = result.data.data.uptime;

          // managers
          this.managers = result.data.data.managers;

          var status = 0;
          var managers = Object.keys(result.data.data.managers);
          for (var mgindex in managers) {
            var manager_key = managers[mgindex];
            var manager = result.data.data.managers[manager_key];

            if (manager_key in this.managers) {
              this.managers[manager_key].error = manager.error;
              this.managers[manager_key].shard_groups = manager.shard_groups;
              this.managers[manager_key].gateway = manager.gateway;
            }

            var shardgroups = Object.values(manager.shard_groups);
            if (shardgroups.length > 0) {
              status = shardgroups.slice(-1)[0].status;
            }
            if (manager.error != "") {
              status = 7;
            }
            this.managers[manager_key].status = status;
          }

          // analytics
          this.analytics = result.data.data.analytics;

          let up = 0;
          let total = 0;
          let guilds = 0;
          this.analytics.colour = "bg-success";

          managers = Object.values(this.analytics.managers);
          for (mgindex in managers) {
            manager = managers[mgindex];
            guilds += manager.guilds;
            shardgroups = Object.values(manager.status);
            for (var sgindex in shardgroups) {
              var shardgroupstatus = shardgroups[sgindex];
              if (2 < shardgroupstatus && shardgroupstatus < 4) {
                up++;
              }
              total++;
            }
          }

          this.analytics.visible = guilds;
          this.analytics.online = up + "/" + total;

          // resttunnel
          if (result.data.data.rest_tunnel_enabled) {
            this.resttunnel.charts = result.data.data.resttunnel.data.charts;
            this.resttunnel.uptime = result.data.data.resttunnel.data.uptime;
            this.resttunnel.numbers = result.data.data.resttunnel.data.numbers;
          }
        })
        .catch(error => {
          if (error.response?.status == 403) {
            clearInterval(this.fetch_task);
          }
          this.showToast("Exception fetching manager data", error);
        })
        .finally(() => {
          this.loading = false;
          this.loadingAnalytics = false;
          this.loadingRestTunnel = false;
        });
    },
    fetchConfiguration() {
      axios
        .get("/api/configuration")
        .then(result => {
          if (result.status == 403) {
            clearInterval(this.fetch_task);
          }
          this.configuration = result.data.data.configuration;
          this.rest_tunnel_enabled = result.data.data.rest_tunnel_enabled;
          this.error = !result.data.success;
        })
        .catch(error => {
          if (error.response?.status == 403) {
            clearInterval(this.fetch_task);
          }
          this.showToast("Exception fetching configuration", error);
        })
        .finally(() => {
          this.loading = false;
        });
    },
    calculateAverage(manager) {
      var totalShards = 0;
      var totalLatency = 0;

      var shardgroups = Object.values(manager.shard_groups);
      for (var sgindex in shardgroups) {
        var shardgroup = shardgroups[sgindex];
        if (shardgroup.status < 6) {
          var shards = Object.values(shardgroup.shards);
          for (var shindex in shards) {
            var shard = shards[shindex];
            var latency =
              new Date(shard.last_heartbeat_ack) -
              new Date(shard.last_heartbeat_sent);
            if (latency > 0) {
              totalLatency += latency;
              totalShards += 1;
            }
          }
        }
      }
      return Math.round(totalLatency / totalShards) || "-";
    },
    calculateAverageShardGroup(shardgroup) {
      var totalShards = 0;
      var totalLatency = 0;

      var shards = Object.values(shardgroup.shards);
      for (var shindex in shards) {
        var shard = shards[shindex];
        var latency =
          new Date(shard.last_heartbeat_ack) -
          new Date(shard.last_heartbeat_sent);
        if (latency > 0) {
          totalLatency += latency;
          totalShards += 1;
        }
      }
      return Math.round(totalLatency / totalShards) || "-";
    },

    since(_uptime) {
      var uptime = new Date(_uptime);
      var difference = (new Date().getTime() - uptime) / 1000;

      var output = "";
      if (difference > 86400) {
        var days = Math.trunc(difference / 86400);
        if (days > 0) {
          output += days + "d";
        }
        difference = difference % 86400;
      }
      if (difference > 3600) {
        var hours = Math.trunc(difference / 3600);
        if (hours > 0) {
          output += hours + "h";
        }
        difference = difference % 3600;
      }
      if (difference > 60) {
        var minutes = Math.trunc(difference / 60);
        if (minutes > 0) {
          output += minutes + "m";
        }
        var seconds = Math.ceil(difference % 60);
        if (seconds > 0) {
          output += seconds + "s";
        }
      }
      return output;
    }
  }
};
</script>

<style scoped>
.visually-hidden {
  clip: rect(0 0 0 0);
  clip-path: inset(50%);
  height: 1px;
  overflow: hidden;
  position: absolute;
  white-space: nowrap;
  width: 1px;
}
</style>
