package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"net/http"
	"strconv"
	"strings"
)


type bnApi struct {
	Price float64 `json:"price,string"`
	Code  int64   `json:"code"`
}

type wallet map[string]float64

var db = map[int64]wallet{}

//var chatId tgbotapi.Update = update.Message.Chat.ID

//запрашиваю актуальную стоимссть валюты
func usdPrice(symbol string) (price float64, err error) {
	resp, err := http.Get(fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%sUSDT", symbol))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var jsonApi bnApi

	err = json.NewDecoder(resp.Body).Decode(&jsonApi)
	if err != nil {
		return
	}

	if jsonApi.Code != 0 {
		err = errors.New("неверный символ")
	}
	price = jsonApi.Price

	return
}
func rubPrice() (price float64, err error) {
	resp, err := http.Get("https://api.binance.com/api/v3/ticker/price?symbol=USDTRUB")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var jsonApi bnApi

	err = json.NewDecoder(resp.Body).Decode(&jsonApi)
	if err != nil {
		return
	}

	price = jsonApi.Price

	return
}

func main() {
	bot, err := tgbotapi.NewBotAPI("TOKEN")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		words := strings.Split(update.Message.Text, " ")
		command := toUpperSlice(words)
		//log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		switch command[0] {
		case "DEL":
			if len(command) != 2 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "неверная команда"))
				break
			}
			//проверяем, если валюта в кошельке, удаляем
			if _, ok := db[update.Message.Chat.ID][command[1]]; ok {
				delete(db[update.Message.Chat.ID], command[1])
				msg := fmt.Sprintf("валюта %s удалена из вашего кошелька", command[1])
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
			} else {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "валюта не найдена"))
			}

		case "ADD":
			if len(command) != 3 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "неверная команда"))
				break
			}

			amount, err := strconv.ParseFloat(command[2], 64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				break
			}

			if _, ok := db[update.Message.Chat.ID]; !ok {
				db[update.Message.Chat.ID] = wallet{}
			}

			db[update.Message.Chat.ID][command[1]] += amount

			balanceText := fmt.Sprintf("Вы добавили: %f %s", db[update.Message.Chat.ID][command[1]], command[1])
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, balanceText))

		case "SUB":
			if len(command) != 3 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				break
			}

			amount, err := strconv.ParseFloat(command[2], 64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "команда не найдена"))
			}

			if _, ok := db[update.Message.Chat.ID]; !ok {
				continue
			}

			if db[update.Message.Chat.ID][command[1]] < amount {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "недостаточно средств для этой операции"))
			} else if db[update.Message.Chat.ID][command[1]] == amount {
				delete(db[update.Message.Chat.ID], command[1])
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "криптовалюта удалена"))
		}else {
				db[update.Message.Chat.ID][command[1]] -= amount

				balanceText := fmt.Sprintf("На счету: %f %s", db[update.Message.Chat.ID][command[1]], command[1])
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, balanceText))
			}

		case "SHOW":
			var msg string
			var sum float64
			rub, _ := rubPrice()

			for key, value := range db[update.Message.Chat.ID] {
				price, _ := usdPrice(key)
				sum += value * price
				msg += fmt.Sprintf("Валюта: %s, сумма: %f, в $: [%.2f]\n", key, value, value*price)

			}
			//вывод общей суммы в двух валютах
			msg += fmt.Sprintf("***===***\nВсего в $: %.2f\nВсего в ₽: %.2f", sum, sum*rub)

			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		case "HELP":
			help := fmt.Sprintf("команды вводятся в формате: КОМАНДА ВАЛЮТА СУММА\n ADD: добавить сумму \n SUB: вычесть сумму \n SHOW: показать баланс \n DEL: удалить валюту")
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, help))

		default:
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "команда не найдена"))

		}

		//msg := tgbotapi.NewMessage(update.Message.Chat.ID, command[0])
		//msg.ReplyToMessageID = update.Message.MessageID

		//bot.Send(msg)
	}
}

//исправляем команды введенных с маленькой буквы
func toUpperSlice(words []string) (command []string) {
	for _, w := range words {
		command = append(command, strings.ToUpper(w))
	}
	return
}
