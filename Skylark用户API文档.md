查询当前空间下所有用户
GET /api/v4/users HTTP/1.1
Authorization: your_authorization

HTTP/1.1 200 OK
[
  {
    "id": 1,
    "name": "第一个管理员",
    "nickname": "",
    "phone": "1111111111",
    "identifier": "1111111111",
    "qq": "123",
    "headimgurl": "/non-digested-assets/avatars/default.png",
    "openid": "123",
    "imported_alias": "第一个管理员",
    "from_wechat": false,
    "tags": [{
      "id": 1,
      "name": "b1",
      "full_name": "bq-b1",
      "color": "#e91e63",
      "tag_group_id": 1,
      "taggings_count": 9
    }, {
      "id": 3,
      "name": "a1",
      "full_name": "a1",
      "color": "#ff5722",
      "tag_group_id": 1,
      "taggings_count": 4
    }, {
      "id": 8,
      "name": "a2",
      "full_name": "a2",
      "color": "#673ab7",
      "tag_group_id": 1,
      "taggings_count": 1
    }
    },
  {
    "id": 1,
    "name": "第二个管理员",
    "nickname": "",
    "phone": "12222222222",
    "identifier": "12222222222",
    "qq": "123",
    "headimgurl": "/non-digested-assets/avatars/default.png",
    "openid": "123",
    "imported_alias": "第一个管理员",
    "from_wechat": false,
    "tags": [{
      "id": 1,
      "name": "b1",
      "full_name": "bq-b1",
      "color": "#e91e63",
      "tag_group_id": 1,
      "taggings_count": 9
    }, {
      "id": 3,
      "name": "a1",
      "full_name": "a1",
      "color": "#ff5722",
      "tag_group_id": 1,
      "taggings_count": 4
    }, {
      "id": 8,
      "name": "a2",
      "full_name": "a2",
      "color": "#673ab7",
      "tag_group_id": 1,
      "taggings_count": 1
    }
  }
]
GET /api/v4/users

所在组织
GET /api/v4/users/14473/organizations HTTP/1.1
Authorization: your_authorization
{
  "name": "name",
  "description": "description",
  "application_strategy": "must_approved",
  "founder_id": 3,
  "patent_id": 13
}
HTTP/1.1 200 OK
[
  {
    "id": 14473,
    "name": "child_1",
    "description": "",
    "description_text": "",
    "created_at": "2017-04-18T16:33:07.408+08:00",
    "updated_at": "2017-12-06T17:52:27.556+08:00",
    "children_count": 1,
    "parent_id": 14472,
    "ancestry": "14472"
  },
  {
    "id": 1935,
    "name": "测试组织",
    "description": "<p>aaa</p>",
    "description_text": "aaa",
    "created_at": "2015-05-25T15:58:45.083+08:00",
    "updated_at": "2018-04-06T15:37:24.712+08:00",
    "children_count": 2,
    "parent_id": null,
    "ancestry": null
  },
  ...
]
GET /api/v4/users/:id/organizations

Parameters

Name	Type	Description	Comments
id	integer	用户id	
查询用户自定义属性
GET /api/v4/user_profile/schema HTTP/1.1
Authorization: your_authorization

HTTP/1.1 200 OK
{
    "id": 1,
    "namespace_id": 1,
    "fields": [
        {
            "id": 6058,
            "title": "测试字段",
            "description": null,
            "type": "Field::TextField",
            "position": 1,
            "validations": [],
            "other_option": null,
            "visibility": "public_visibility",
            "marked": false,
            "settings": {
                "layout": "block",
                "register_writed": false,
                "explicit": false,
                "enable_modify": true,
                "char_size_limit_settings": {},
                "other_option_settings": {
                    "limit": {}
                },
                "length_limit": {},
                "limit_settings": {},
                "enable_wechat_scan": false
            },
            "detail_id": null,
            "identity_key": "daa8ac34837a4ce4a826d2b68f6d446d",
            "data": {}
        },
        {
            "id": 6035,
            "title": "街道选择",
            "description": null,
            "type": "Field::CascadedSelect",
            "position": 0,
            "validations": [],
            "other_option": null,
            "visibility": "public_visibility",
            "marked": false,
            "settings": {
                "layout": "block",
                "register_writed": true,
                "explicit": false,
                "enable_modify": true,
                "other_option_settings": {
                    "limit": {}
                },
                "length_limit": {},
                "limit_settings": {},
                "char_size_limit_settings": {}
            },
            "detail_id": null,
            "identity_key": "203ed7d1e1d545179f9d889f106d0e98",
            "data": {},
            "cascaded_select": {
                "id": 72,
                "name": "高新服务所在地区",
                "labels": [
                    "省份",
                    "市",
                    "区",
                    "街办"
                ],
                "locked": true,
                "min_label_length": 4,
                "choices": [
                    {
                        "id": 7647,
                        "name": "四川省",
                        "ancestry": null,
                        "parent_id": null,
                        "locked": true,
                        "children_count": 1
                    }
                ]
            }
        }
    ]
}
GET api/v4/user_profile/schema

查询用户信息
GET /api/v4/users/1 HTTP/1.1
Authorization: your_authorization

HTTP/1.1 200 OK
{
  "id": 1,
  "name": "高新服务小助手",
  "nickname": "坤",
  "sex": 0,
  "phone": "18980807092",
  "identifier": "18980807092",
  "openid": "oyyT9s3lmVAnPV9CwYHL2zFLMf4E",
  "created_at": "2020-07-13T16:27:44.749+08:00",
  "updated_at": "2022-05-22T21:10:16.405+08:00",
  "headimgurl": "http://thirdwx.qlogo.cn/mmopen/vi_32/Ok2zIF8CS9AP28EWv75Ox9uVahUVtWN21ic3IxMILCpudNUl59Gcia94WtqU8aLz4t5PIpT9F84sYIMBr5oGDLGw/96",
  "tags": [
    {
      "id": 18,
      "name": "可视化"
    }
  ],
  "organizations": [
    {
      "id": 1,
      "name": "政务服务处",
      "description": "<p>您可以根据需要更改名字及描述，或者删除组织</p>",
      "created_at": "2020-07-13T16:27:44.800+08:00"
    },
    {
      "id": 3,
      "name": "社区治理",
      "description": "",
      "created_at": "2020-09-04T15:36:32.953+08:00"
    },
    {
      "id": 5,
      "name": "街办数据统计",
      "description": "",
      "created_at": "2020-09-17T17:37:25.962+08:00"
    },
    {
      "id": 18,
      "name": "授权记录",
      "description": null,
      "created_at": "2021-04-22T14:00:22.511+08:00"
    }
  ],
  "profile_submission": {
    "entries": [
      {
        "id": 401385,
        "field_id": 9632,
        "option_id": null,
        "value": "81",
        "choice_id": null,
        "value_id": null,
        "latitude": null,
        "longitude": null,
        "group_id": null,
        "detail_id": null
      },
      {
        "id": 55527,
        "field_id": 9633,
        "option_id": null,
        "value": "81",
        "choice_id": null,
        "value_id": null,
        "latitude": null,
        "longitude": null,
        "group_id": null,
        "detail_id": null
      },
      {
        "id": 1,
        "field_id": 64,
        "option_id": null,
        "value": "四川省-成都市-高新区-中和街办",
        "choice_id": 6,
        "value_id": null,
        "latitude": null,
        "longitude": null,
        "group_id": null,
        "detail_id": null
      }
    ]
  }
}
GET /api/v4/users/:id

Parameters

Name	Type	Description	Comments
id	integer	用户id	
创建用户
POST /api/v4/users/ HTTP/1.1
Authorization: your_authorization

HTTP/1.1 200 OK
{
  "id": 1,
  "name": "第一个管理员",
  "nickname": "",
  "phone": "1111111111",
  "identifier": "1111111111",
  "qq": "123",
  "headimgurl": "/non-digested-assets/avatars/default.png",
  "openid": "123",
  "imported_alias": "第一个管理员",
  "from_wechat": false,
  "tags": [{
    "id": 1,
    "name": "b1",
    "full_name": "bq-b1",
    "color": "#e91e63",
    "tag_group_id": 1,
    "taggings_count": 9
  }, {
    "id": 3,
    "name": "a1",
    "full_name": "a1",
    "color": "#ff5722",
    "tag_group_id": 1,
    "taggings_count": 4
  }, {
    "id": 8,
    "name": "a2",
    "full_name": "a2",
    "color": "#673ab7",
    "tag_group_id": 1,
    "taggings_count": 1
  }
}
POST /api/v4/users

Parameters

Name	Type	Description	Comments
name	String	名字	
identifier	String	识别码	
phone	String	手机号	
openid	String	微信的 openid	
更新用户
PUT /api/v4/users/:id HTTP/1.1
Authorization: your_authorization
{
    "identifier": "18828072450",
    "name": "name",
    "phone": "18828072450",
    "profile_submission_attributes": {
        "entries_attributes": [
            {
                "id": 1143157,
                "field_id": 6058,
                "value": "test"
            },
            {
                "id": 1091241,
                "field_id": 6035,
                "choice_id": 7650,
                "value": "四川省-成都市-高新区-桂溪街办"
            }
        ]
    }
}

HTTP/1.1 200 OK
{
    "id": 79,
    "name": "name",
    "nickname": "k k",
    "sex": 0,
    "phone": "18828072450",
    "identifier": "18828072450",
    "openid": "oXDm5s2v7fnJ2Mut-UiiHtCEIb6Q",
    "created_at": "2017-02-22T13:56:54.125+08:00",
    "updated_at": "2022-11-21T14:29:26.011+08:00",
    "headimgurl": "https://thirdwx.qlogo.cn/mmopen/vi_32/Q0j4TwGTfTKFLWWPT1sSVywib8qpNNfLjMOliblYqa105ibCOKGvzwRV0vAEAlGLcwXBliboaad9udfsHcl7SKIfOw/96",
    "tags": [],
    "organizations": [
        {
            "id": 619,
            "name": "22秋",
            "description": "",
            "created_at": "2022-06-14T15:36:42.148+08:00"
        }
    ],
    "profile_submission": {
        "entries": [
            {
                "id": 1143157,
                "field_id": 6058,
                "option_id": null,
                "value": "test",
                "choice_id": null,
                "value_id": null,
                "latitude": null,
                "longitude": null,
                "group_id": null,
                "detail_id": null
            },
            {
                "id": 1091241,
                "field_id": 6035,
                "option_id": null,
                "value": "四川省-成都市-高新区-桂溪街办",
                "choice_id": 7650,
                "value_id": null,
                "latitude": null,
                "longitude": null,
                "group_id": null,
                "detail_id": null,
                "choice": {
                    "id": 7650,
                    "name": "桂溪街办",
                    "ancestry": "7647/7648/7649",
                    "parent_id": 7649,
                    "locked": true,
                    "children_count": 0
                }
            }
        ]
    }
}
PUT /api/v4/users/:id

Parameters

Name	Type	Description	Comments
id	integer	用户 id	
name	String	名字	
identifier	String	识别码	
phone	String	手机号	
openid	String	微信的 openid	
更新用户标签
POST /api/v4/users/:id/tags HTTP/1.1
Authorization: your_authorization
HTTP/1.1 200 OK
{
  "id": 1,
  "name": "第一个管理员",
  "nickname": "",
  "phone": "1111111111",
  "identifier": "1111111111",
  "qq": "123",
  "headimgurl": "/non-digested-assets/avatars/default.png",
  "openid": "123",
  "imported_alias": "第一个管理员",
  "from_wechat": false,
  "tags": [{
    "id": 1,
    "name": "b1",
    "full_name": "bq-b1",
    "color": "#e91e63",
    "tag_group_id": 1,
    "taggings_count": 9
  }, {
    "id": 3,
    "name": "a1",
    "full_name": "a1",
    "color": "#ff5722",
    "tag_group_id": 1,
    "taggings_count": 4
  }, {
    "id": 8,
    "name": "a2",
    "full_name": "a2",
    "color": "#673ab7",
    "tag_group_id": 1,
    "taggings_count": 1
  }
}
POST /api/v4/users/:id/tags

Parameters

Name	Type	Description	Comments
id	integer	用户 id	
add_tag_ids	integer[]	需要添加标签的 ids	
remove_tag_ids	integer[]	需要删除的标签 ids	
查询用户人脸核身信息
GET /api/v4/users/:id/identity_certification HTTP/1.1
Authorization: your_authorization
HTTP/1.1 200 OK
{
 "id": 1,
 "data": {}
}
GET /api/v4/users/:id/identity_certification

Parameters

Name	Type	Description	Comments
id	integer	用户 id	
通过关键字搜索用户
GET /api/v4/users/query HTTP/1.1
Authorization: your_authorization
HTTP/1.1 200 OK
[
  {
    "id": 1,
    "name": "第一个管理员",
    "nickname": "",
    "phone": "1111111111",
    "identifier": "1111111111",
    "qq": "123",
    "headimgurl": "/non-digested-assets/avatars/default.png",
    "openid": "123",
    "imported_alias": "第一个管理员",
    "from_wechat": false,
    "tags": [{
      "id": 1,
      "name": "b1",
      "full_name": "bq-b1",
      "color": "#e91e63",
      "tag_group_id": 1,
      "taggings_count": 9
    }, {
      "id": 3,
      "name": "a1",
      "full_name": "a1",
      "color": "#ff5722",
      "tag_group_id": 1,
      "taggings_count": 4
    }, {
      "id": 8,
      "name": "a2",
      "full_name": "a2",
      "color": "#673ab7",
      "tag_group_id": 1,
      "taggings_count": 1
    }
  }
]
GET /api/v4/users/query

Parameters

Name	Type	Description	Comments
keyword	string	名字或手机号	
