package store

import (
	"github.com/fernandez14/spartangeek-blacker/modules/mail"
	"github.com/fernandez14/spartangeek-blacker/modules/user"
	"github.com/fernandez14/spartangeek-blacker/modules/helpers"
	"gopkg.in/mgo.v2/bson"
	"time"
	"strings"
)

type One struct {
	di   *Module
	data *OrderModel
}

func (self *One) Data() *OrderModel {

	return self.data
}

func (self *One) PushAnswer(text, kind string) {

	if kind != "text" && kind != "note" {

		return
	}

	database := self.di.Mongo.Database

	message := MessageModel{
		Content: text,
		Type:    kind,
		Created: time.Now(),
		Updated: time.Now(),
	}

	err := database.C("orders").Update(bson.M{"_id": self.data.Id}, bson.M{"$push": bson.M{"messages": message}, "$set": bson.M{"updated_at": time.Now()}})

	if err != nil {
		panic(err)
	}

	if kind == "text" {

		// Send an email async
		go func() {

			mailing := self.di.Mail
			text = strings.Replace(text, "\n", "<br>", -1)
			
			compose := mail.Mail{
				Template: 250241,
				Recipient: []mail.MailRecipient{
					{
						Name:  self.data.User.Name,
						Email: self.data.User.Email,
					},
				},
				FromEmail: "pc@spartangeek.com",
				FromName: "Drak Spartan",
				Variables: map[string]interface{}{
					"content": text,
				},
			}

			mailing.Send(compose)
		}()
	}
}

func (self *One) PushTag(tag string) {

	database := self.di.Mongo.Database
	item := TagModel{
		Name: tag,
		Created: time.Now(),
	}

	err := database.C("orders").Update(bson.M{"_id": self.data.Id}, bson.M{"$push": bson.M{"tags": item}})

	if err != nil {
		panic(err)
	}
}

func (self *One) PushActivity(name, description string, due_at time.Time) {

	database := self.di.Mongo.Database
	activity := ActivityModel{
		Name: name,
		Description: description,
		Done: false,
		Due: due_at,
		Created: time.Now(),
		Updated: time.Now(),
	}

	err := database.C("orders").Update(bson.M{"_id": self.data.Id}, bson.M{"$push": bson.M{"activities": activity}})

	if err != nil {
		panic(err)
	}
}

func (self *One) PushInboundAnswer(text string, mail bson.ObjectId) {

	database := self.di.Mongo.Database

	message := MessageModel{
		Content: text,
		Type:    "inbound",
		RelatedId: mail,
		Created: time.Now(),
		Updated: time.Now(),
	}

	err := database.C("orders").Update(bson.M{"_id": self.data.Id}, bson.M{"$push": bson.M{"messages": message}, "$set": bson.M{"unreaded": true, "updated_at": time.Now()}})

	if err != nil {
		panic(err)
	}
}

func (self *One) Stage(name string) {

	// Temp way to validate the name of the stage
	if name != "estimate" && name != "negotiation" && name != "accepted" && name != "awaiting" && name != "closed" {
		return
	}

	database := self.di.Mongo.Database

	// Define steps in order
	steps := []string{"estimate", "negotiation", "accepted", "awaiting", "closed"}
	current := self.data.Pipeline.Step

	if current > 0 {
		current = current-1
	}

	target := 0

	for index, step := range steps {

		if step == name {

			target = index
		}
	}

	named := steps[target]
	err := database.C("orders").Update(bson.M{"_id": self.data.Id}, bson.M{"$set": bson.M{"pipeline.step": target+1, "pipeline.current": named, "pipeline.updated_at": time.Now(), "updated_at": time.Now()}})

	if err != nil {
		panic(err)
	}
}

func (self *One) MatchUsers() []user.UserBasic {

	database := self.di.Mongo.Database
	ip := self.data.User.Ip

	if ip != "" {

		var checkins []user.CheckinModel
		var users_id []bson.ObjectId

		err := database.C("checkins").Find(bson.M{"client_ip": ip}).All(&checkins)

		if err != nil {
			panic(err)
		}

		for _, checkin := range checkins {

			duplicated, _ := helpers.InArray(checkin.UserId, users_id)

			if ! duplicated {

				users_id = append(users_id, checkin.UserId)
			}
		}

		var users []user.UserBasic

		err = database.C("users").Find(bson.M{"$or": []bson.M{
				{"_id": bson.M{"$in": users_id}},
				{"email": self.data.User.Email},
				{"facebook.email": self.data.User.Email},
			}}).Select(bson.M{"_id": 1, "username": 1, "username_slug": 1, "email": 1, "facebook": 1, "validated": 1, "banned": 1, "created_at": 1, "updated_at": 1}).All(&users)

		if err != nil {
			panic(err)
		}

		return users
	} 

	var users []user.UserBasic

	err := database.C("users").Find(bson.M{"$or": []bson.M{
			{"email": self.data.User.Email},
			{"facebook.email": self.data.User.Email},
		}}).Select(bson.M{"_id": 1, "username": 1, "username_slug": 1, "email": 1, "facebook": 1, "validated": 1, "banned": 1, "created_at": 1, "updated_at": 1}).All(&users)

	if err != nil {
		panic(err)
	}

	return users

}

func (self *One) Touch() {

	database := self.di.Mongo.Database
	
	err := database.C("orders").Update(bson.M{"_id": self.data.Id}, bson.M{"$set": bson.M{"unreaded": false}})

	if err != nil {
		panic(err)
	}
}