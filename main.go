package main

import (
    "fmt"
    "log"
    "time"
    "errors"
    "strings"
    "runtime"
    "os/exec"
    "github.com/parnurzeal/gorequest"
    "github.com/tidwall/gjson"
)

var modelMarks = map[string]string{
    "iphone12pro": "A",
    "iphone12promax": "G",
}

var storeCityMap map[string]map[string]map[string]string
var storeNumberMap map[string]map[string]string

var selectedCity string = "北京"
var selectedModel string = "iphone12pro 128gb 石墨色"

func getModels() []string {
    return []string{
        "iphone12pro 128gb 石墨色-MGL93CH/A",
        "iphone12pro 128gb 银色-MGLA3CH/A",
        "iphone12pro 128gb 金色-MGLC3CH/A",
        "iphone12pro 128gb 海蓝色-MGLD3CH/A",
        "iphone12pro 256gb 石墨色-MGLE3CH/A",
        "iphone12pro 256gb 银色-MGLF3CH/A",
        "iphone12pro 256gb 金色-MGLG3CH/A",
        "iphone12pro 256gb 海蓝色-MGLH3CH/A",
        "iphone12pro 512gb 石墨色-MGLJ3CH/A",
        "iphone12pro 512gb 银色-MGLK3CH/A",
        "iphone12pro 512gb 金色-MGLL3CH/A",
        "iphone12pro 512gb 海蓝色-MGLM3CH/A",
        "iphone12promax 128gb 石墨色-MGC03CH/A",
        "iphone12promax 128gb 银色-MGC13CH/A",
        "iphone12promax 128gb 金色-MGC23CH/A",
        "iphone12promax 128gb 海蓝色-MGC33CH/A",
        "iphone12promax 256gb 石墨色-MGC43CH/A",
        "iphone12promax 256gb 银色-MGC53CH/A",
        "iphone12promax 256gb 金色-MGC63CH/A",
        "iphone12promax 256gb 海蓝色-MGC73CH/A",
        "iphone12promax 512gb 石墨色-MGC93CH/A",
        "iphone12promax 512gb 银色-MGCA3CH/A",
        "iphone12promax 512gb 金色-MGCC3CH/A",
        "iphone12promax 512gb 海蓝色-MGCE3CH/A",
    }
}

func getModelMap() map[string]string {
    models := getModels()
    modelMap := make(map[string]string, len(models))

    for _, model := range models {
        kv := strings.Split(model, "-")
        modelMap[kv[0]] = kv[1]
    }

    return modelMap
}

func getModelCodeMap() map[string]string {
    modelMap := getModelMap()
    modelCodeMap := make(map[string]string, len(modelMap))
    for model, modelCode := range modelMap {
        modelCodeMap[modelCode] = model
    }

    return modelCodeMap
}

func getModel(modelCode string) string {
    return getModelCodeMap()[modelCode]
}

func getModelCode(model string) string {
    return getModelMap()[model]
}

func getModelMark(model string) string {
    return modelMarks[strings.Split(model, " ")[0]]
}

func printDoc() {
    fmt.Println(`
        本程序会自动进行库存查询，库存为空时，会自动查找同系列有库存机型，
        用户可选择使用该同系列机型注册码（同系列机型注册码通用， 30分钟有效）。
        待指定型号有库存时，会自动跳转到预约页面，输入该注册码即可。
    `)
}

func initSelected() {
    fmt.Println("可预约城市列表")

    citySlice := make([]string, 0, len(storeCityMap))
    cityIndex := 0
    for city, _ := range storeCityMap {
        fmt.Printf("    [%d] %s\n", cityIndex, city)

        citySlice = append(citySlice, city)
        cityIndex++
    }

SELECTCITY:
    fmt.Print("请选择要预约的城市【输入前缀编号即可】：")
    fmt.Scanln(&cityIndex)
    if cityIndex < 0 || cityIndex >= len(citySlice) {
        fmt.Println("选择有误，请重新选择！")
        goto SELECTCITY
    }

    selectedCity = citySlice[cityIndex]
    fmt.Println("您选择的城市为：", selectedCity)

    fmt.Println("可预约型号列表")

    models := getModels()
    modelIndex := 0
    for modelIndex, model := range getModels() {
        fmt.Printf("    [%d] %s\n", modelIndex, model)
    }

SELECTMODEL:
    fmt.Print("请选择要预约的型号【输入前缀编号即可】：")
    fmt.Scanln(&modelIndex)
    if modelIndex < 0 || modelIndex >= len(models) {
        fmt.Println("选择有误，请重新选择！")
        goto SELECTMODEL
    }

    selectedModel = models[modelIndex]
    fmt.Println("您选择的型号为：", selectedModel)
}

func initStores() {
    storeCityMap = make(map[string]map[string]map[string]string)
    storeNumberMap = make(map[string]map[string]string)

    _, body, errs := gorequest.New().Get("https://reserve-prime.apple.com/CN/zh_CN/reserve/A/stores.json").End()
    if len(errs) != 0 {
        log.Fatalln(errs[0].Error())
    }

    for _, store := range gjson.Get(body, "stores").Array() {
        city := store.Get("city").String()
        storeName := store.Get("storeName").String()
        storeNumber := store.Get("storeNumber").String()
        if _, ok := storeCityMap[city]; !ok {
            storeCityMap[city] = make(map[string]map[string]string)
        }

        storeMap := map[string]string{
            "city": city,
            "name": storeName,
            "number": storeNumber,
        }

        storeCityMap[city][storeNumber] = storeMap
        storeNumberMap[storeNumber] = storeMap
    }
}

func getAvailability(model string, cityStores map[string]map[string]string) (string, string) {
    availabilityUrl := "https://reserve-prime.apple.com/CN/zh_CN/reserve/" + getModelMark(model) + "/availability.json"
    _, body, errs := gorequest.New().Get(availabilityUrl).End()
    if len(errs) != 0 {
        return "", ""
    }

    for storeNumber := range cityStores {
        availability := gjson.Get(body, "stores." + storeNumber + "." + getModelCode(model) + ".availability")
        if availability.Map()["contract"].Bool() && availability.Map()["unlocked"].Bool() {
            return storeNumber, strings.Split(model, "-")[1]
        } else {
            log.Println(selectedCity, cityStores[storeNumber]["name"], selectedModel, "无货")
        }
    }

    log.Println("---------------------------------------------------------------------------")

    return "", ""
}

// https://reserve-prime.apple.com/CN/zh_CN/reserve/A?quantity=1&anchor-store=R390&store=R390&partNumber=MGLJ3CH/A&plan=unlocked
func getAnyoneAvailability(model string, storeNumberMap map[string]map[string]string) error {
    _, body, errs := gorequest.New().Get("https://reserve-prime.apple.com/CN/zh_CN/reserve/" + modelMarks[strings.Split(model, " ")[0]] + "/availability.json").End()
    if len(errs) != 0 {
        return errs[0]
    }

    url := ""
    for storeNumber, modelsAvailability := range gjson.Get(body, "stores").Map() {
        for modelCode, modelAvailability := range modelsAvailability.Map(){
            if modelAvailability.Get("availability.contract").Bool() && modelAvailability.Get("availability.unlocked").Bool() {
                model := getModel(modelCode)
                if url == "" {
                    url = getReserveUrlByCodeMark(getModelMark(model), modelCode, storeNumber)

                    log.Printf("先获取注册号（可能为同型号其他颜色/存储）：%s %s\n", model, url)
                }

                store := storeNumberMap[storeNumber]
                log.Printf("%s %s-%s %s 有货\n", store["city"], store["number"], store["name"], model)
            }
        }
    }

    if url != "" {
        openBrowser(url)
        return nil
    }

    return errors.New("所有门店均无货")
}

func getReserveUrlByModel(model string, storeNumber string) string {
    return getReserveUrlByCodeMark(getModelMark(model), getModelCode(model), storeNumber)
}

func getReserveUrlByCodeMark(modelMark string, modelCode string, storeNumber string) string {
    return "https://reserve-prime.apple.com/CN/zh_CN/reserve/" + modelMark + "?quantity=1&anchor-store=" +
        storeNumber + "&store=" + storeNumber + "&partNumber=" + modelCode + "&plan=unlocked"
}

func openBrowser(url string) {
    var err error
    switch runtime.GOOS {
    case "linux":
        err = exec.Command("xdg-open", url).Start()
    case "windows":
        err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
    case "darwin":
        err = exec.Command("open", url).Start()
    default:
        err = errors.New("不支持该操作系统")
    }
    if err != nil {
        log.Fatalln(errors.New("打开网页失败，请自行手动操作" + url))
    }
}

func main() {
    printDoc()

    initStores()

    initSelected()

    log.Printf("开始执行预约程序，城市【%s】，型号【%s】\n", selectedCity, selectedModel)

    go func() {
        timerFunc := func() {
            err := getAnyoneAvailability(selectedModel, storeNumberMap)
            if err != nil {
                log.Println(err)
            }
        }

        timerFunc()

        for {
            select {
            case <- time.After(time.Minute * 28):
                timerFunc()
            }
        }

    }()

    for {
        time.Sleep(time.Second * 2)

        storeNumber, modelCode := getAvailability(selectedModel, storeCityMap[selectedCity])
        if storeNumber != "" && modelCode != "" {
            url := getReserveUrlByModel(selectedModel, modelCode)

            fmt.Println(url)

            openBrowser(url)

            break
        }
    }
}
