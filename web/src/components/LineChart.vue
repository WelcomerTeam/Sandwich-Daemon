<script>
import { mixins, Line } from "vue-chartjs";
// import "chartjs-adapter-moment";

export default {
  name: "LineChart",
  extends: Line,
  mixins: [mixins.reactiveProp],
  props: ["chartData", "options"],
  mounted() {
    this.renderChart(this.chartData, this.options);
  }
};
</script>
