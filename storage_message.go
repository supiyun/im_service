package main
import "bytes"
import "encoding/binary"
import log "github.com/golang/glog"



//存储服务器消息
const MSG_SAVE_AND_ENQUEUE = 200
const MSG_DEQUEUE = 201
const MSG_LOAD_OFFLINE = 202
const MSG_RESULT = 203

//内部文件存储使用
const MSG_OFFLINE = 254
const MSG_ACK_IN = 255


func init() {
	message_creators[MSG_SAVE_AND_ENQUEUE] = func()IMessage{return new(SAEMessage)}
	message_creators[MSG_DEQUEUE] = func()IMessage{return new(OfflineMessage)}
	message_creators[MSG_LOAD_OFFLINE] = func()IMessage{return new(AppUserID)}
	message_creators[MSG_RESULT] = func()IMessage{return new(MessageResult)}
	message_creators[MSG_OFFLINE] = func()IMessage{return new(OfflineMessage)}
	message_creators[MSG_ACK_IN] = func()IMessage{return new(OfflineMessage)}

	message_descriptions[MSG_SAVE_AND_ENQUEUE] = "MSG_SAVE_AND_ENQUEUE"
	message_descriptions[MSG_DEQUEUE] = "MSG_DEQUEUE"
	message_descriptions[MSG_LOAD_OFFLINE] = "MSG_LOAD_OFFLINE"
	message_descriptions[MSG_RESULT] = "MSG_RESULT"

}

type EMessage struct {
	msgid int64
	msg   *Message
}

type OfflineMessage struct {
	appid    int64
	receiver int64
	msgid    int64
}
type DQMessage OfflineMessage 

func (off *OfflineMessage) ToData() []byte {
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, off.appid)
	binary.Write(buffer, binary.BigEndian, off.receiver)
	binary.Write(buffer, binary.BigEndian, off.msgid)
	buf := buffer.Bytes()
	return buf
}

func (off *OfflineMessage) FromData(buff []byte) bool {
	if len(buff) < 24 {
		return false
	}
	buffer := bytes.NewBuffer(buff)
	binary.Read(buffer, binary.BigEndian, &off.appid)
	binary.Read(buffer, binary.BigEndian, &off.receiver)
	binary.Read(buffer, binary.BigEndian, &off.msgid)
	return true
}


type SAEMessage struct {
	msg       *Message
	receivers []*AppUserID
}

func (sae *SAEMessage) ToData() []byte {
	if sae.msg == nil {
		return nil
	}

	if sae.msg.cmd == MSG_SAVE_AND_ENQUEUE {
		log.Warning("recusive sae message")
		return nil
	}

	buffer := new(bytes.Buffer)
	mbuffer := new(bytes.Buffer)
	SendMessage(mbuffer, sae.msg)
	msg_buf := mbuffer.Bytes()
	var l int16 = int16(len(msg_buf))
	binary.Write(buffer, binary.BigEndian, l)
	buffer.Write(msg_buf)
	var count int16 = int16(len(sae.receivers))
	binary.Write(buffer, binary.BigEndian, count)
	for _, r := range(sae.receivers) {
		binary.Write(buffer, binary.BigEndian, r.appid)
		binary.Write(buffer, binary.BigEndian, r.uid)
	}
	buf := buffer.Bytes()
	return buf
}

func (sae *SAEMessage) FromData(buff []byte) bool {
	if len(buff) < 4 {
		return false
	}

	buffer := bytes.NewBuffer(buff)
	var l int16
	binary.Read(buffer, binary.BigEndian, &l)
	if int(l) > buffer.Len() {
		return false
	}

	msg_buf := make([]byte, l)
	buffer.Read(msg_buf)
	mbuffer := bytes.NewBuffer(msg_buf)
	//recusive
	msg := ReceiveMessage(mbuffer)
	if msg == nil {
		return false
	}
	sae.msg = msg

	if buffer.Len() < 2 {
		return false
	}
	var count int16
	binary.Read(buffer, binary.BigEndian, &count)
	if buffer.Len() < int(count)*16 {
		return false
	}
	sae.receivers = make([]*AppUserID, count)
	for i := int16(0); i < count; i++ {
		r := &AppUserID{}
		binary.Read(buffer, binary.BigEndian, &r.appid)
		binary.Read(buffer, binary.BigEndian, &r.uid)
		sae.receivers[i] = r
	}
	return true
}

type MessageResult struct {
	status int32
	content []byte
}
func (result *MessageResult) ToData() []byte {
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, result.status)
	buffer.Write(result.content)
	buf := buffer.Bytes()
	return buf
}

func (result *MessageResult) FromData(buff []byte) bool {
	if len(buff) < 4 {
		return false
	}

	buffer := bytes.NewBuffer(buff)
	binary.Read(buffer, binary.BigEndian, &result.status)
	result.content = buff[4:]
	return true
}