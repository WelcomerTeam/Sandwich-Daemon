import quart
app = quart.Quart(__name__)


@app.route("/", methods=["GET", "POST"])
async def index():
    return await quart.send_file("test.html")

app.run()
