import re
import os

# Converts structures with json to also include a msgpack copy converting
# GuildID string \`msgpack:"guild_id" msgpack:"guild_id" json:"guild_id" msgpack:"guild_id"\`
# into
# GuildID string \`msgpack:"guild_id" msgpack:"guild_id" json:"guild_id" msgpack:"guild_id" msgpack:"guild_id"\`
# and will ignore ones that have already got msgpack

a = "json"
b = "yaml"
c = "msgpack"
rgx = re.compile(f"`(({a}|{b}|{c}):\"(\S+)\"\s?)+`")


def convert(string):
    done = []
    results = re.findall(rgx, string)
    for result in results:
        # print(result)
        try:
            index = result.index("json")

            of = list(filter(bool, result[::3]))
            oline = " ".join(of)

            _final = result[::3]
            _final += f'msgpack:"{result[index+1]}"',

            stones = {}
            for val in _final:
                stone, value = val.split(":")
                stones[stone] = value
            _final = list(
                filter(bool, [f"{stone}:{value}" for stone, value in stones.items()]))
            line = " ".join(_final)

            string = string.replace("`"+oline+"`", "`"+line+"`")
            print(oline, "=>", line)

        except Exception as e:
            print(e)
            return

        # string = string.replace(line, " ".join(list(set(line.split(" ")))))
        # print(line, "=>", " ".join(list(set(line.split(" ")))))

    return string


def scan(dir):
    print(dir)
    for f in os.listdir(dir):
        path = os.path.join(dir, f)
        if os.path.isdir(path) and not ".git" in f:
            print("")
            scan(path)
        if os.path.isfile(path):
            try:
                print("-->", path)
                file = open(path, "r")
                res = convert(file.read())
                file.close()

                file = open(path, "w")
                file.write(res)
                file.close()
            except Exception as e:
                print(e)


scan(os.getcwd())
