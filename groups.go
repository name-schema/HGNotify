package main

import (
	"fmt"
	"strings"

	"github.com/go-yaml/yaml"
)

type (
	GroupList map[string]*namedGroup
)

type namedGroup struct {
	ID            uint     `json:"id" yaml:"id"`
	Name          string   `json:"groupName" yaml:"groupName"`
	Members       []Member `json:"members" yaml:"members"`
	isPrivate     bool
	privacyRoomID string
}

type Member struct {
	Name string `json:"memberName" yaml:"memberName"`
	GID  string `json:"gchatID" yaml:"gchatID"`
}

func (ng *namedGroup) addMember(member User) {
	addition := Member{
		Name: member.Name,
		GID:  member.GID,
	}

	ng.Members = append(ng.Members, addition)
}

func (ng *namedGroup) removeMember(member User) {
	for i, groupMember := range ng.Members {
		if member.GID == groupMember.GID {
			ng.Members = append(ng.Members[:i], ng.Members[i+1:]...)
		}
	}
}

func (gl GroupList) Create(groupName string, msgObj messageResponse) string {
	saveName, meta := gl.CheckGroup(groupName, msgObj)
	if meta != "" {
		if strings.Contains(meta, "exist") {
			return fmt.Sprintf("Group %q seems to already exist.\nIf you'd like to remove and recreate the group please say \"%s delete %s\" followed by \"%s create %s @Members...\"", groupName, BOTNAME, groupName, BOTNAME, groupName)
		}
	}

	var (
		mentions   = msgObj.Message.Mentions
		newGroup   = new(namedGroup)
		newMembers string

		seen = checkSeen()
	)

	newGroup.Name = groupName
	newGroup.ID = gl.getID()
	newGroup.isPrivate = false

	for i, mention := range mentions {
		if seen(mention.Called.Name) {
			continue
		}

		if mention.Called.Type != "BOT" && mention.Type == "USER_MENTION" {
			if i > 1 {
				newMembers += ","
			}
			newGroup.addMember(mention.Called.User)

			newMembers += " " + mention.Called.Name
		}
	}

	gl[saveName] = newGroup
	return fmt.Sprintf("Created group %q with user(s) %s", groupName, newMembers)
}

func (gl GroupList) Delete(groupName string, msgObj messageResponse) string {
	saveName, meta := gl.CheckGroup(groupName, msgObj)
	if meta != "" {
		if !strings.Contains(meta, "exist") {
			return fmt.Sprintf("Group %q does not seem to exist.", groupName)
		}

		if strings.Contains(meta, "private") {
			return fmt.Sprintf("The group %q is private, and you may not mutate it.", groupName)
		}
	}

	delete(gl, saveName)
	return fmt.Sprintf("Group %q has been deleted, along with all it's data.", groupName)
}

func (gl GroupList) AddMembers(groupName string, msgObj messageResponse) string {
	saveName, meta := gl.CheckGroup(groupName, msgObj)
	if meta != "" {
		if !strings.Contains(meta, "exist") {
			return fmt.Sprintf("Group %q does not seem to exist.", groupName)
		}

		if strings.Contains(meta, "private") {
			return fmt.Sprintf("The group %q is private, and you may not mutate it.", groupName)
		}
	}

	var (
		addedMembers    string
		existingMembers string
		text            string

		seen = checkSeen()
	)

	for _, mention := range msgObj.Message.Mentions {
		if seen(mention.Called.Name) {
			continue
		}

		if mention.Called.Type != "BOT" && mention.Type == "USER_MENTION" {
			exist := gl.CheckMember(groupName, mention.Called.GID)

			if !exist {
				gl[saveName].addMember(mention.Called.User)

				addedMembers += mention.Called.Name + " "
			} else {
				existingMembers += mention.Called.Name + " "
			}
		}
	}

	if addedMembers != "" {
		text += fmt.Sprintf("Got [ %s] added to the group %q. ", addedMembers, groupName)
	}

	if existingMembers != "" {
		text += fmt.Sprintf("\nUser(s) [ %s] already added the group %q. ", existingMembers, groupName)
	}

	return text
}

func (gl GroupList) RemoveMembers(groupName string, msgObj messageResponse) string {
	saveName, meta := gl.CheckGroup(groupName, msgObj)
	if meta != "" {
		if !strings.Contains(meta, "exist") {
			return fmt.Sprintf("Group %q does not seem to exist.", groupName)
		}

		if strings.Contains(meta, "private") {
			return fmt.Sprintf("The group %q is private, and you may not mutate it.", groupName)
		}
	}

	var (
		removedMembers     string
		nonExistantMembers string
		text               string

		seen = checkSeen()
	)

	for _, mention := range msgObj.Message.Mentions {
		if seen(mention.Called.Name) {
			continue
		}

		if mention.Called.Type != "BOT" && mention.Type == "USER_MENTION" {
			exist := gl.CheckMember(groupName, mention.Called.GID)

			if exist {
				gl[saveName].removeMember(mention.Called.User)

				removedMembers += mention.Called.Name + " "
			} else {
				nonExistantMembers += mention.Called.Name + " "
			}
		}
	}

	if removedMembers != "" {
		text += fmt.Sprintf("I've removed [ %s] from %q. ", removedMembers, groupName)
	}

	if nonExistantMembers != "" {
		text += fmt.Sprintf("\nUser(s) [ %s] didn't seem to exist when attempting to remove them from %q. ", nonExistantMembers, groupName)
	}

	return text
}

func (gl GroupList) Restrict(groupName string, msgObj messageResponse) string {
	saveName, meta := gl.CheckGroup(groupName, msgObj)
	if !strings.Contains(meta, "exist") {
		return fmt.Sprintf("Group %q does not seem to exist.", groupName)
	}

	if strings.Contains(meta, "private") {
		return fmt.Sprintf("The group %q is private, and you may not mutate it.", groupName)
	}

	if gl[saveName].isPrivate {
		gl[saveName].isPrivate = false
		gl[saveName].privacyRoomID = ""

		return fmt.Sprintf("I've set %q to public, now it can be used in any room.", groupName)
	}

	gl[saveName].isPrivate = true
	gl[saveName].privacyRoomID = msgObj.Room.GID
	return fmt.Sprintf("I've set %q to be private, the group can only be used in this room now.", groupName)
}

func (gl GroupList) Notify(groupName string, msgObj messageResponse) string {
	saveName, meta := gl.CheckGroup(groupName, msgObj)
	if !strings.Contains(meta, "exist") {
		return fmt.Sprintf("Group %q does not seem to exist.", groupName)
	}

	if strings.Contains(meta, "private") {
		return fmt.Sprintf("The group %q is private, and you may not use it.", groupName)
	}

	var memberList string
	for _, member := range gl[saveName].Members {
		memberList += "<" + member.GID + "> "
	}

	message := msgObj.Message.Text

	botLen := len(BOTNAME)
	botIndex := strings.Index(message, BOTNAME)

	tmpMessage := string([]byte(message)[botLen+botIndex:])

	groupLen := len(groupName)
	groupIndex := strings.Index(tmpMessage, groupName)

	newMessage := fmt.Sprintf("%s said:\n\n%s",
		msgObj.Message.Sender.Name,
		strings.Replace(
			message,
			string([]byte(message)[botIndex:botIndex+botLen+groupIndex+groupLen]),
			memberList,
			1,
		),
	)

	if len(newMessage) >= 4000 {
		return "My apologies, your message with the group added would exceed Google Chat's character limit. :("
	}

	return newMessage
}

func (gl GroupList) List(groupName string, msgObj messageResponse) string {
	if groupName == "" {
		if len(gl) == 0 {
			return fmt.Sprint("There are no groups to show currently. :(")
		}

		var allGroupNames string
		for name := range gl {
			_, meta := gl.CheckGroup(name, msgObj)
			if !strings.Contains(meta, "private") {
				allGroupNames += " | " + gl[name].Name
			}
		}

		return fmt.Sprintf("Here are all of the usable group names: ```%s``` If the group is private, it will not appear in this list. Ask me about a specfic group for more information. ( %s list groupName )", string([]byte(allGroupNames)[3:]), BOTNAME)
	}

	saveName, meta := gl.CheckGroup(groupName, msgObj)
	if meta != "" {
		if !strings.Contains(meta, "exist") {
			return fmt.Sprintf("Group %q does not seem to exist.", groupName)
		}

		if strings.Contains(meta, "private") {
			return fmt.Sprintf("The group %q is private, and you may not view it.", groupName)
		}
	}

	yamlList, err := yaml.Marshal(gl[saveName])
	checkError(err)

	return fmt.Sprintf("Here are details for %q: ```%s```", groupName, string(yamlList))
}

func (gl GroupList) CheckGroup(groupName string, msgObj messageResponse) (saveName, meta string) {
	//TODO: Check for proper formatting
	saveName = strings.ToLower(groupName)
	group, exist := gl[saveName]

	if exist {
		meta += "exist"
	} else {
		return
	}

	if group.isPrivate {
		if group.privacyRoomID != msgObj.Room.GID {
			meta += "private"
		}
	}

	return
}

func (gl GroupList) CheckMember(groupName, memberID string) (here bool) {
	saveName := strings.ToLower(groupName)

	if len(gl[saveName].Members) == 0 {
		here = false
	} else {
		here = true
	}

	for _, member := range gl[saveName].Members {
		if memberID == member.GID {
			here = true
			break
		} else {
			here = false
		}
	}

	return
}

func (gl GroupList) getID() uint {
	if len(gl) == 0 {
		return uint(1)
	}

	id := uint(0)
	for _, group := range gl {
		highestID := group.ID
		if highestID > id {
			id = highestID + 1
		}
	}

	return id
}

func checkSeen() func(name string) bool {
	var seenMembers []string

	return func(name string) bool {
		for _, seenMember := range seenMembers {
			if seenMember == name {
				return true
			}
		}

		seenMembers = append(seenMembers, name)
		return false
	}
}
