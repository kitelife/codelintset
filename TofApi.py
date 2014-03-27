#coding: utf-8

import httplib
import datetime
import time
import re
import cgi


class TofApi(object):

    def __init__(self):
        self.app_key = '38d7860df0bb41f2be9622959ac98955'
        self.host = 'ws.xxx.oa.com'
        self.post = '/MessageService.svc'
        self.envolope_template = '''
        <s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
            <s:Header>
                <Application_Context xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
                    <AppKey xmlns="http://schemas.datacontract.org/2004/07/Tencent.OA.Framework.Context">{app_key}</AppKey>
                </Application_Context>
            </s:Header>
            <s:Body>
                {body}
            </s:Body>
        </s:Envelope>
        '''

    def _fill_envolope(self, envolope_body_template, *data):
        envolope_body = envolope_body_template %data
        self.envolope = self.envolope_template.format(app_key=self.app_key, body=envolope_body)

    def _send_msg(self):
        web_service = httplib.HTTP(self.host)
        web_service.putrequest("POST", self.post)
        web_service.putheader("Host", self.host)
        web_service.putheader("User-Agent", "Python Post")
        web_service.putheader("Content-type", 'text/xml; charset="UTF-8"')
        web_service.putheader("Content-length", "%d" % len(self.envolope))
        web_service.putheader("SOAPAction", self.soap_action)
        web_service.endheaders()
        web_service.send(self.envolope)

        status_code, status_message, header = web_service.getreply()
        response_content = web_service.getfile().read()
        matchers = re.search('\<SendRTXResult\>([^\<\>]*)\<\/SendRTXResult\>', response_content)

        if status_code == 200 and matchers and matchers.group(1) == 'true':
            return True
        else:
            return False

    def send_rtx(self, sender, receiver, title, msg):
        self.soap_action = 'http://tempuri.org/IMessageService/SendRTX'
        envolope_body_template = '''
        <SendRTX xmlns="http://tempuri.org/">
            <message xmlns:d4p1="http://schemas.datacontract.org/2004/07/Tencent.OA.Framework.Messages.DataContract" xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
                <d4p1:MsgInfo>%s</d4p1:MsgInfo>
                <d4p1:Priority>Normal</d4p1:Priority>
                <d4p1:Receiver>%s</d4p1:Receiver>
                <d4p1:Sender>%s</d4p1:Sender>
                <d4p1:Title>%s</d4p1:Title>
            </message>
        </SendRTX>
        '''
        data = (cgi.escape(msg), receiver, sender, cgi.escape(title))
        self._fill_envolope(envolope_body_template, *data)
        return self._send_msg()

    def send_mail(self, sender, receiver, title, msg):
        self.soap_action = 'http://tempuri.org/IMessageService/SendMail'

        start_time = datetime.datetime.now().isoformat()
        end_time = datetime.datetime(2019, 12, 25).isoformat()

        envolope_body_template = '''
        <SendMail xmlns="http://tempuri.org/">
            <mail xmlns:d4p1="http://schemas.datacontract.org/2004/07/Tencent.OA.Framework.Messages.DataContract" xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
                <d4p1:Attachments i:nil="true" />
                <d4p1:Bcc i:nil="true" />
                <d4p1:BodyFormat>Text</d4p1:BodyFormat>
                <d4p1:CC i:nil="true" />
                <d4p1:Content>%s</d4p1:Content>
                <d4p1:EmailType>SEND_TO_ENCHANGE</d4p1:EmailType>
                <d4p1:EndTime>%s</d4p1:EndTime>
                <d4p1:From>%s</d4p1:From>
                <d4p1:Location i:nil="true" />
                <d4p1:MessageStatus>Pickup</d4p1:MessageStatus>
                <d4p1:Organizer i:nil="true" />
                <d4p1:Priority>Normal</d4p1:Priority>
                <d4p1:StartTime>%s</d4p1:StartTime>
                <d4p1:Title>%s</d4p1:Title>
                <d4p1:To>%s</d4p1:To>
            </mail>
        </SendMail>
        '''
        data = (cgi.escape(msg), end_time, sender, start_time, cgi.escape(title), receiver)
        self._fill_envolope(envolope_body_template, *data)
        return self._send_msg()

    def send_sms(self, sender, receiver, msg):
        self.soap_action = 'http://tempuri.org/IMessageService/SendSMS'

        envolope_body_template = '''
        <SendSMS xmlns="http://tempuri.org/">
            <message xmlns:d4p1="http://schemas.datacontract.org/2004/07/Tencent.OA.Framework.Messages.DataContract" xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
                <d4p1:MsgInfo>%s</d4p1:MsgInfo>
                <d4p1:Priority>Normal</d4p1:Priority>
                <d4p1:Receiver>%s</d4p1:Receiver>
                <d4p1:Sender>%s</d4p1:Sender>
            </message>
        </SendSMS>
        '''
        data = (cgi.escape(msg), receiver, sender)
        self._fill_envolope(envolope_body_template, *data)
        return self._send_msg()


if __name__ == '__main__':

    api = TofApi()

    print api.send_rtx('yongfengxia', 'yongfengxia', 'Test&<', "test>>&")
    print api.send_mail('xxx@email.com', 'xxx@email.com', 'Test', 'test')
    print api.send_sms('yongfengxia', 'xxxxxxxxxxx', 'test')
