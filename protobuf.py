import os


def proto(filepath, filename):
    # print("from", filepath, "to protobuf/", filename, ".proto")
    os.system(f"go2proto -f protobuf -p {filepath}")
    os.system(f"mv protobuf/output.proto protobuf/{filename}.proto")


folder = "./discord/structs"

for f in os.listdir(folder):
    filename = os.path.splitext(f)[0]
    proto(os.path.join(folder, f), filename)
