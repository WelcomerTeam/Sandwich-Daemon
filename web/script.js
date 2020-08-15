vue = new Vue({
    el: '#app',
    data() {
        return {
            data: {
                data: {}
            },
            loading: true,
        }
    },
    mounted() {
        axios
            .get('/api/configuration.json')
            .then(response => (this.data = response))
            .catch(error => console.log(error))
            .finally(() => this.loading = false)
    }
})