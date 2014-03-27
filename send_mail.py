#!/usr/bin/python
#coding: utf-8

import argparse
from TofApi import TofApi

def get_argparser():
    parser = argparse.ArgumentParser()
    parser.add_argument("-s", action="store", default=None, help="mail sender")
    parser.add_argument("-r", action="store", default=None, help="mail receiver")
    parser.add_argument("-t", action="store", default=None, help="mail title")
    parser.add_argument("-c", action="store", default=None, help="mail content")
    
    return parser

def main():
    parser = get_argparser()
    args = parser.parse_args()
    
    sender = args.s
    receiver = args.r
    title = args.t
    content = args.c
    #print sender, receiver, title, content
    if sender and receiver and title and content:
        api = TofApi()
        api.send_mail(sender, receiver, title, content)

if __name__ == '__main__':
    main()
