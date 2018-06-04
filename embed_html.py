
"""
    This python script will combine all the frontend files(.js, .html, .css) in ./web
    to one go source file (web.go)
"""
import os
valid_files = [".js", ".html", ".css"]
content = ""
index = {}
def generate_go(nouse, dir, files):
    def valid(f):
        for v in valid_files:
            if f.endswith(v):
                return True
        return False

    def getOneFile(f):
        global content
        ff = open(dir+os.sep+f)
        i0 = len(content)
        fcontent = ff.read()
        i1 = len(fcontent) + i0
        print(f)
        content += fcontent
        index[dir[len("web"):]+"/"+f] = (i0, i1)

    for f in filter(valid, files):
        getOneFile(f)

if __name__ == "__main__":
    outf = open("web/web.go", "w")
    outf.write("//auto generated - don't edit it\n")
    outf.write("package web\n")
    outf.write('import "errors"\n')
    os.path.walk("web", generate_go, 0)
    outf.write("var content = []byte(`" + content.replace('`', '`+"`"+`') + "`)\n")
    outf.write("""type contentIndexStruct struct {
    begin int
    end int
}
""")
    outf.write("var contentIndex = map[string]contentIndexStruct{")
    for k in index:
        v = index[k]
        outf.write('"' + k + '":{' + str(v[0]) + ',' + str(v[1]) + '},\n')
    outf.write("}\n")

    outf.write("""func GetContent(uri string) ([]byte, error) {
    if val, ok := contentIndex[uri]; ok {
        return content[val.begin:val.end], nil
    }
    return []byte{}, errors.New("not found")
}""")





