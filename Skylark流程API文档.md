Flows
获取流程详情
GET /api/v4/yaw/flows/:id HTTP/1.1
Authorization: your_authorization

HTTP/1.1 200 OK
{
  "id":8,
  "title":"简单流程",
  "fields":[
    {
      "id":38761,
      "title":"别填",
      "description":null
    },
    {
      "id":38806,
      "title":"明细清单",
      "description":null
    }
  ],
  "vertices":[
    {
      "id":78,
      "name":"开始节点",
      "type":"YetAnotherWorkflow::Vertex::Initial"
    },
    {
      "id":80,
      "name":"流程节点3",
      "type":"YetAnotherWorkflow::Vertex::Normal"
    },
    {
      "id":79,
      "name":"结束节点",
      "type":"YetAnotherWorkflow::Vertex::Final"
    },
    {
      "id":90,
      "name":"流程节点2",
      "type":"YetAnotherWorkflow::Vertex::Normal"
    },
    {
      "id":91,
      "name":"流程节点1",
      "type":"YetAnotherWorkflow::Vertex::Normal"
    },
    {
      "id":113,
      "name":"开始节点",
      "type":"YetAnotherWorkflow::Vertex::Initial"
    },
    {
      "id":114,
      "name":"结束节点",
      "type":"YetAnotherWorkflow::Vertex::Final"
    }
  ],
  "edges":[
    {
      "id":101,
      "from_vertex_id":78,
      "to_vertex_id":91
    },
    {
      "id":92,
      "from_vertex_id":80,
      "to_vertex_id":79
    },
    {
      "id":103,
      "from_vertex_id":90,
      "to_vertex_id":80
    },
    {
      "id":102,
      "from_vertex_id":91,
      "to_vertex_id":90
    }
  ]
}

GET /api/v4/yaw/flows/:id

Parameters

Name	Type	Description	Comments
id	integer	流程id	
获取发起的流程任务列表
GET /api/v4/yaw/flows/:id/journeys HTTP/1.1
Authorization: your_authorization

HTTP/1.1 200 OK
X-SLP-Current-Page: 1
X-SLP-Total-Pages: 29
X-SLP-Total-Count: 2
[
  {
    "id":110,
    "sn":"820180503184630000029",
    "status":"processing",
    "current_duration_threshold":null,
    "created_at":"2018-06-06T15:39:10.532+08:00",
    "updated_at":"2018-06-06T15:39:10.532+08:00",
    "current_vertex_id":91,
    "reviewer_vertex_ids":[
      78
    ],
    "response":{
      "id":548288,
      "cached_values":{
        "38761":{
          "value":[
            "123123123"
          ],
          "text_value":[
            "123123123"
          ],
          "exported_value":[
            "123123123"
          ]
        }
      }
    },
    "user":{
      "id":17013,
      "name":"aaaa",
      "identifier":"12345",
      "headimgurl":"/non-digested-assets/avatars/default.png"
    }
  },
  ...
]
GET /api/v4/yaw/flows/:id/journeys

Parameters

Name	Type	Description	Comments
id	integer	流程id	
获取单条流程记录
GET /api/v4/yaw/flows/:id/journeys/:journey_id HTTP/1.1
Authorization: your_authorization

HTTP/1.1 200 OK
{
    "id": 830,
    "sn": "146920210421115723000002",
    "status": "processing",
    "current_duration_threshold": null,
    "flow_id": 1469,
    "created_at": "2022-06-20T10:42:48.008+08:00",
    "updated_at": "2022-06-20T10:42:48.008+08:00",
    "current_vertex_id": 5323,
    "reviewer_vertex_ids": [
        5317
    ],
    "journey_url": "https://zwfw.cdht.gov.cn/namespaces/1/yet_another_workflow/journeys/830",
    "response": {
        "id": 2959,
        "cached_values": {
            "8978": {
                "value": [
                    {
                        "id": 2701,
                        "gid": "gid://skylark/Option/2701",
                        "value": "新选项2"
                    }
                ],
                "text_value": [
                    "新选项2"
                ],
                "exported_value": [
                    "新选项2"
                ]
            },
            "8979": {
                "value": [
                    "2022-06-20"
                ],
                "text_value": [
                    "2022-06-20"
                ],
                "exported_value": [
                    "2022-06-20"
                ]
            },
            "8980": {
                "value": [
                    {
                        "id": 9027,
                        "gid": "gid://skylark/Attachment/9027",
                        "value": "飞书20220304-205019.jpg"
                    }
                ],
                "text_value": [
                    "飞书20220304-205019.jpg"
                ],
                "exported_value": [
                    "飞书20220304-205019.jpg（https://zwfw.cdht.gov.cn/attachments/9027/download）"
                ]
            },
            "11720": {
                "value": [
                    "test"
                ],
                "text_value": [
                    "test"
                ],
                "exported_value": [
                    "test"
                ]
            }
        },
        "mapped_values": {
            "55090a52efe748ed87da86cdaa34d123": {
                "value": [
                    {
                        "id": 2701,
                        "gid": "gid://skylark/Option/2701",
                        "value": "新选项2"
                    }
                ],
                "text_value": [
                    "新选项2"
                ],
                "exported_value": [
                    "新选项2"
                ]
            },
            "2157bad17e7b45a5bcc572eac17fa8c0": {
                "value": [
                    "2022-06-20"
                ],
                "text_value": [
                    "2022-06-20"
                ],
                "exported_value": [
                    "2022-06-20"
                ]
            },
            "fa908beece324b31955141147bff6b66": {
                "value": [
                    {
                        "id": 9027,
                        "gid": "gid://skylark/Attachment/9027",
                        "value": "飞书20220304-205019.jpg"
                    }
                ],
                "text_value": [
                    "飞书20220304-205019.jpg"
                ],
                "exported_value": [
                    "飞书20220304-205019.jpg（https://zwfw.cdht.gov.cn/attachments/9027/download）"
                ]
            },
            "96987d995bb24e92ab982aa312e8b70a": {
                "value": [
                    "test"
                ],
                "text_value": [
                    "test"
                ],
                "exported_value": [
                    "test"
                ]
            }
        },
        "entries": [
            {
                "id": 1893464,
                "field_id": 8979,
                "option_id": null,
                "value": "2022-06-20",
                "choice_id": null,
                "value_id": null,
                "latitude": null,
                "longitude": null,
                "group_id": null,
                "detail_id": null
            },
            {
                "id": 1893463,
                "field_id": 8980,
                "option_id": null,
                "value": "飞书20220304-205019.jpg",
                "choice_id": null,
                "value_id": 9026,
                "latitude": null,
                "longitude": null,
                "group_id": null,
                "detail_id": null,
                "attachment": {
                    "id": 9026,
                    "name": "飞书20220304-205019.jpg",
                    "size": "469288",
                    "mime_type": "image/jpeg",
                    "extension": "image",
                    "extra_info": {},
                    "download_url": "http://fs-attachment.yqfw.cdyoue.com/1-1655692959-455733ae5fcd10af93ec0ef6f3f46d32-1655692943605?attname=%E9%A3%9E%E4%B9%A620220304-205019.jpg&e=1655711959&token=A02msK5084aaUq-gB1MdVJnPnO9g2p9jD87oMMPg:K4nA0k3JRrdodn7LDkfFYgbAOyc="
                }
            },
            {
                "id": 1893462,
                "field_id": 8978,
                "option_id": 2701,
                "value": "新选项2",
                "choice_id": null,
                "value_id": null,
                "latitude": null,
                "longitude": null,
                "group_id": null,
                "detail_id": null
            },
            {
                "id": 1893461,
                "field_id": 11720,
                "option_id": null,
                "value": "test",
                "choice_id": null,
                "value_id": null,
                "latitude": null,
                "longitude": null,
                "group_id": null,
                "detail_id": null
            }
        ]
    },
    "user": {
        "id": 1,
        "name": "高新服务小助手",
        "nickname": "坤",
        "sex": 0,
        "phone": "180000000",
        "identifier": "180000000",
        "openid": "oyyT9s3lmVAnPV9CwYHL2zFLMf4E",
        "created_at": "2020-07-13T16:27:44.749+08:00",
        "updated_at": "2022-05-22T21:10:16.405+08:00",
        "headimgurl": "http://thirdwx.qlogo.cn/mmopen/vi_32/Ok2zIF8CS9AP28EWv75Ox9uVahUVtWN21ic3IxMILCpudNUl59Gcia94WtqU8aLz4t5PIpT9F84sYIMBr5oGDLGw/96"
    }
}
GET /api/v4/yaw/flows/:id/journeys/:journey_id

Name	Type	Description	Comments
id	integer	流程id	
journey_id	integer	流程记录id	
获取某条流程任务的所有 Moment(处理情况)
GET /api/v4/yaw/journeys/:id/moments HTTP/1.1
Authorization: your_authorization

HTTP/1.1 200 OK
[
  {
    "id": 392,
    "status": "proposed",
    "comment": null,
    "esignature": "data:image/png;base64,something",
    "created_at": "2018-06-29T15:09:31.835+08:00",
    "updated_at": "2018-06-29T15:09:31.835+08:00",
    "vertex_id": 168,
    "user": {
      "id": 127057,
      "name": "cacao",
      "identifier": null,
      "headimgurl": "/non-digested-assets/avatars/default.png"
    }
  },
  {
    "id": 393,
    "status": "approved",
    "comment": "",
    "esignature": "data:image/png;base64,something",
    "created_at": "2018-06-29T15:09:46.297+08:00",
    "updated_at": "2018-06-29T15:09:46.297+08:00",
    "vertex_id": 170,
    "user": {
      "id": 127057,
      "name": "cacao",
      "identifier": null,
      "headimgurl": "/non-digested-assets/avatars/default.png"
    }
  }
]
GET /api/v4/yaw/journeys/:id/moments

Parameters

Name	Type	Description	Comments
id	integer	流程id	
获取流程的所有 vertices (节点)
GET /api/v4/yaw/flows/:flow_id/vertices/:id HTTP/1.1
Authorization: your_authorization

HTTP/1.1 200 OK
{
    "id": 23881,
    "name": "天全社区",
    "type": "YetAnotherWorkflow::Vertex::Normal",
    "alias_name": "",
    "fields": [],
    "user_boundary": {
        "id": 94384,
        "organization_ids": [
            277
        ],
        "user_tag_ids": [],
        "user_tagged_type": "intersection",
        "only_binded": false,
        "overlord_type": "YetAnotherWorkflow::Vertex",
        "overlord_id": 23881,
        "created_at": "2022-07-18T19:39:31.914+08:00",
        "updated_at": "2022-08-18T17:22:13.212+08:00",
        "namespace_id": 1,
        "user_ids": [],
        "status": "default",
        "append_count": 0,
        "cached_organization_ids": [
            277
        ],
        "cached_user_ids": [
            76,
            104
        ],
        "extra_boundary": {
            "reviewer_vertex_ids": []
        },
        "type": "UserBoundary::VertexReview"
    }
}
GET /api/v4/yaw/flows/:flow_id/vertices/:id

Parameters

Name	Type	Description	Comments
flow_id	integer	流程id	
id	integer	节点id
Vertex Column	Meaning
user_boundary	
cached_user_ids	固定节点处理人
extra_boundary	动态节点处理人
创建流程任务（route）
POST /api/v4/yaw/flows/:id/journeys HTTP/1.1
Authorization: your_authorization
{
    "assignment": {
        "operation": "route",
        "response_attributes": {
            "entries_attributes": [
                {
                    "field_id": 7199,
                    "value": "test"
                },
                {
                    "field_id": 7200,
                    "value": "新选项",
                    "option_id": 8854
                },
                {
                    "field_id": 7201,
                    "value": "2020-11-18"
                },
                {
                    "field_id": 7202,
                    "value": "WechatIMG132.jpeg",
                    "value_id": 3839,
                    "_destroy": false
                }
            ]
        }
    },
    "user_id": 6,
    "webhook": {
      "payload_url": "",
      "subscribed_events": [
        "JourneyStatusEvent",
      ],
    },
}
HTTP/1.1 200 OK
{
    "assignment": {
        "id": 7802,
        "status": "stashed",
        "category": "proposed",
        "read": true,
        "current_duration_threshold": null,
        "created_at": "2020-11-18T15:20:08.113+08:00",
        "updated_at": "2020-11-18T15:45:24.343+08:00",
        "after_submit_action": "default",
        "after_submit_redirect_url": null,
        "gid": "gid://skylark/YetAnotherWorkflow::Assignment/7802",
        "pretty_current_duration_threshold": null,
        "journey": {
            "id": 2236,
            "sn": "145920201118152008000001",
            "custom_sn": null,
            "status": "stashed",
            "current_duration_threshold": null,
            "created_at": "2020-11-18T15:20:08.076+08:00",
            "updated_at": "2020-11-18T15:20:08.076+08:00",
            "gid": "gid://skylark/YetAnotherWorkflow::Journey/2236",
            "pretty_current_duration_threshold": null,
            "response": {
                "id": 78149,
                "created_at": "2020-11-18T15:20:08.063+08:00",
                "entries": []
            },
            "user": {
                "id": 79,
                "name": "樊翔宇",
                "nickname": "k k",
                "phone": "180000000",
                "identifier": "180000000",
                "qq": null,
                "headimgurl": "https://thirdwx.qlogo.cn/mmopen/vi_32/Q0j4TwGTfTKFLWWPT1sSVywib8qpNNfLjMOliblYqa105ibCOKGvzwRV0vAEAlGLcwXia8AK9FQiavG1tDxcCCkfUCA",
                "openid": "oXDm5s2v7fnJ2Mut-UiiHtCEIb6Q",
                "imported_alias": "樊翔宇",
                "gid": "gid://skylark/User/79"
            }
        },
        "response": {
            "id": 78150,
            "created_at": "2020-11-18T15:20:08.097+08:00",
            "entries": [
                {
                    "id": 1141143,
                    "field_id": 7202,
                    "option_id": null,
                    "value": "WechatIMG132.jpeg",
                    "choice_id": null,
                    "value_id": 3839,
                    "latitude": null,
                    "longitude": null,
                    "group_id": null,
                    "detail_id": null,
                    "attachment": {
                        "id": 3839,
                        "name": "WechatIMG132.jpeg",
                        "size": "16998",
                        "mime_type": "image/jpeg",
                        "extension": "image",
                        "extra_info": {}
                    }
                },
                {
                    "id": 1141142,
                    "field_id": 7201,
                    "option_id": null,
                    "value": "2020-11-18",
                    "choice_id": null,
                    "value_id": null,
                    "latitude": null,
                    "longitude": null,
                    "group_id": null,
                    "detail_id": null
                },
                {
                    "id": 1141141,
                    "field_id": 7200,
                    "option_id": 8854,
                    "value": "新选项",
                    "choice_id": null,
                    "value_id": null,
                    "latitude": null,
                    "longitude": null,
                    "group_id": null,
                    "detail_id": null
                },
                {
                    "id": 1141140,
                    "field_id": 7199,
                    "option_id": null,
                    "value": "test",
                    "choice_id": null,
                    "value_id": null,
                    "latitude": null,
                    "longitude": null,
                    "group_id": null,
                    "detail_id": null
                }
            ]
        }
    },
    "next_vertices": [
        {
            "id": 5600,
            "name": "流程节点",
            "type": "YetAnotherWorkflow::Vertex::Normal",
            "metadata": {
                "position": {
                    "x": 397,
                    "y": 251.703125
                }
            },
            "operations": {
                "route": {
                    "name": "选路",
                    "enabled": true
                },
                "refuse": {
                    "name": "回退",
                    "enabled": true
                },
                "approve": {
                    "name": "通过",
                    "enabled": true
                },
                "comment": {
                    "name": "处理意见",
                    "enabled": true
                },
                "transfer": {
                    "name": "转交",
                    "enabled": true
                },
                "esignature": {
                    "name": "电子签字",
                    "enabled": false
                }
            },
            "settings": {
                "carbon_copy": {
                    "enable_manual": false
                },
                "refuse_mode": 1,
                "assign_manually": false,
                "batch_processing": false,
                "duration_threshold": {
                    "value": null,
                    "enable_delay": false,
                    "business_time": {
                        "final": "17:00",
                        "initial": "09:00",
                        "workday": true
                    },
                    "enable_manual": false,
                    "enable_business_time": false
                },
                "distributed_equally": false
            },
            "graph_id": 1730,
            "created_at": "2020-11-18T15:20:05.493+08:00",
            "updated_at": "2020-11-18T15:20:05.568+08:00",
            "gid": "gid://skylark/YetAnotherWorkflow::Vertex::Normal/5600",
            "fields": []
        }
    ],
    "next_graphs": [
        {
            "id": 1730,
            "name": "主流程",
            "ancestry": null,
            "settings": {
                "visibility": "public",
                "duration_threshold": {
                    "value": null,
                    "enable_delay": false,
                    "business_time": {
                        "final": "17:00",
                        "initial": "09:00",
                        "workday": true
                    },
                    "enable_manual": false,
                    "enable_business_time": false
                }
            },
            "metadata": {},
            "created_at": "2020-11-18T15:03:32.377+08:00",
            "updated_at": "2020-11-18T15:03:32.487+08:00",
            "gid": "gid://skylark/YetAnotherWorkflow::Graph/1730"
        }
    ]
}
POST /api/v4/yaw/flows/:id/journeys

Parameters

Name	Type	Description	Comments
operation	string	寻路固定为 route	
webhook	object	用于流程自动化请求的参数	流程没有自动化可不传
webhook.payload_url	string	可接收post请求的链接 用于监听流程自动化的回调	
webhook.subscribed_events	string	固定值： "JourneyStatusEvent"	
发起流程的完整请求都是固定为先寻路（route），后发起（propose）
发起流程任务（propose）
POST /api/v4/yaw/flows/:id/journeys HTTP/1.1
Authorization: your_authorization
{
    "assignment": {
      "operation": "propose",
      "next_vertex_id": 1,
      "duration_thresholds": [
        { 
          "gid"=>"gid://skylark/YetAnotherWorkflow::Vertex::Normal/7557", 
          "value"=>"2023-07-24 14:29"
        },
        { 
          "gid"=>"gid://skylark/YetAnotherWorkflow::Graph/1079", 
          "value"=>"2023-07-24 14:29"
        }
      ]
    },
    "user_id": 6,
    "webhook": {
      "payload_url": "",
      "subscribed_events": [
        "JourneyStatusEvent",
      ],
    }
}
HTTP/1.1 200 OK
{
    "id": 7802,
    "status": "processing",
    "category": "proposed",
    "read": true,
    "current_duration_threshold": null,
    "created_at": "2020-11-18T16:05:08.207+08:00",
    "updated_at": "2020-11-18T16:05:08.261+08:00",
    "after_submit_action": "default",
    "after_submit_redirect_url": null,
    "gid": "gid://skylark/YetAnotherWorkflow::Assignment/7802",
    "pretty_current_duration_threshold": null,
    "journey": {
        "id": 2236,
        "sn": "145920201118152008000001",
        "custom_sn": null,
        "status": "processing",
        "current_duration_threshold": null,
        "created_at": "2020-11-18T16:05:08.207+08:00",
        "updated_at": "2020-11-18T16:05:08.207+08:00",
        "gid": "gid://skylark/YetAnotherWorkflow::Journey/2236",
        "pretty_current_duration_threshold": null,
        "response": {
            "id": 78149,
            "created_at": "2020-11-18T15:20:08.063+08:00",
            "entries": []
        },
        "user": {
            "id": 79,
            "name": "樊翔宇",
            "nickname": "k k",
            "phone": "180000000",
            "identifier": "180000000",
            "qq": null,
            "headimgurl": "https://thirdwx.qlogo.cn/mmopen/vi_32/Q0j4TwGTfTKFLWWPT1sSVywib8qpNNfLjMOliblYqa105ibCOKGvzwRV0vAEAlGLcwXia8AK9FQiavG1tDxcCCkfUCA",
            "openid": "oXDm5s2v7fnJ2Mut-UiiHtCEIb6Q",
            "imported_alias": "樊翔宇",
            "gid": "gid://skylark/User/79"
        }
    },
    "response": {
        "id": 78150,
        "created_at": "2020-11-18T15:20:08.097+08:00",
        "entries": [
            {
                "id": 1141143,
                "field_id": 7202,
                "option_id": null,
                "value": "WechatIMG132.jpeg",
                "choice_id": null,
                "value_id": 3839,
                "latitude": null,
                "longitude": null,
                "group_id": null,
                "detail_id": null,
                "attachment": {
                    "id": 3839,
                    "name": "WechatIMG132.jpeg",
                    "size": "16998",
                    "mime_type": "image/jpeg",
                    "extension": "image",
                    "extra_info": {}
                }
            },
            {
                "id": 1141142,
                "field_id": 7201,
                "option_id": null,
                "value": "2020-11-18",
                "choice_id": null,
                "value_id": null,
                "latitude": null,
                "longitude": null,
                "group_id": null,
                "detail_id": null
            },
            {
                "id": 1141141,
                "field_id": 7200,
                "option_id": 8854,
                "value": "新选项",
                "choice_id": null,
                "value_id": null,
                "latitude": null,
                "longitude": null,
                "group_id": null,
                "detail_id": null
            },
            {
                "id": 1141140,
                "field_id": 7199,
                "option_id": null,
                "value": "test",
                "choice_id": null,
                "value_id": null,
                "latitude": null,
                "longitude": null,
                "group_id": null,
                "detail_id": null
            }
        ]
    }
}
POST /api/v4/yaw/flows/:id/journeys

Parameters

Name	Type	Description	Comments
operation	string	发起固定为 propose	
next_vertex_id	integer	需要到达的下一个流程节点，从 [发起流程任务（寻路）] 的返回结果获取	
duration_thresholds[gid]	string	设置时限的 gid	流程时限和节点时限用不同格式区分，格式参考 demo
duration_thresholds[value]	string	处理时限	
发起流程的完整请求都是固定为先寻路（route），后发起（propose）
获取流程节点所有处理信息
GET /api/v4/yaw/journeys/1/assignments HTTP/1.1
Authorization: your_authorization
HTTP/1.1 200 OK
[
    {
        "id": 1295,
        "assignee_id": 1,
        "status": "processing",
        "category": "proposed",
        "read": true,
        "operation_data": {
            "operation": "propose",
            "next_vertex_id": 5323
        },
        "current_duration_threshold": null,
        "vertex_id": 5317,
        "journey_id": 830,
        "created_at": "2022-06-20T10:42:48.008+08:00",
        "updated_at": "2022-06-20T10:42:48.109+08:00",
        "response": {
            "id": 2960,
            "created_at": "2021-04-21T11:57:23.102+08:00",
            "cached_values": {
                "8978": {
                    "value": [
                        {
                            "id": 2701,
                            "gid": "gid://skylark/Option/2701",
                            "value": "新选项2"
                        }
                    ],
                    "text_value": [
                        "新选项2"
                    ],
                    "exported_value": [
                        "新选项2"
                    ]
                },
                "8979": {
                    "value": [
                        "2022-06-20"
                    ],
                    "text_value": [
                        "2022-06-20"
                    ],
                    "exported_value": [
                        "2022-06-20"
                    ]
                },
                "8980": {
                    "value": [
                        {
                            "id": 9026,
                            "gid": "gid://skylark/Attachment/9026",
                            "value": "飞书20220304-205019.jpg"
                        }
                    ],
                    "text_value": [
                        "飞书20220304-205019.jpg"
                    ],
                    "exported_value": [
                        "飞书20220304-205019.jpg（https://zwfw.cdht.gov.cn/attachments/9026/download）"
                    ]
                },
                "11720": {
                    "value": [
                        "test"
                    ],
                    "text_value": [
                        "test"
                    ],
                    "exported_value": [
                        "test"
                    ]
                }
            },
            "mapped_values": {
                "55090a52efe748ed87da86cdaa34d123": {
                    "value": [
                        {
                            "id": 2701,
                            "gid": "gid://skylark/Option/2701",
                            "value": "新选项2"
                        }
                    ],
                    "text_value": [
                        "新选项2"
                    ],
                    "exported_value": [
                        "新选项2"
                    ]
                },
                "2157bad17e7b45a5bcc572eac17fa8c0": {
                    "value": [
                        "2022-06-20"
                    ],
                    "text_value": [
                        "2022-06-20"
                    ],
                    "exported_value": [
                        "2022-06-20"
                    ]
                },
                "fa908beece324b31955141147bff6b66": {
                    "value": [
                        {
                            "id": 9026,
                            "gid": "gid://skylark/Attachment/9026",
                            "value": "飞书20220304-205019.jpg"
                        }
                    ],
                    "text_value": [
                        "飞书20220304-205019.jpg"
                    ],
                    "exported_value": [
                        "飞书20220304-205019.jpg（https://zwfw.cdht.gov.cn/attachments/9026/download）"
                    ]
                },
                "96987d995bb24e92ab982aa312e8b70a": {
                    "value": [
                        "test"
                    ],
                    "text_value": [
                        "test"
                    ],
                    "exported_value": [
                        "test"
                    ]
                }
            }
        }
    },
    {
        "id": 1430,
        "assignee_id": 1,
        "status": "processing",
        "category": "processed",
        "read": true,
        "operation_data": {},
        "current_duration_threshold": null,
        "vertex_id": 5323,
        "journey_id": 830,
        "created_at": "2022-06-20T10:42:48.393+08:00",
        "updated_at": "2022-06-20T16:12:15.504+08:00",
        "response": {
            "id": 208895,
            "created_at": "2022-06-20T10:42:48.388+08:00",
            "cached_values": {},
            "mapped_values": {}
        }
    }
]
GET /api/v4/yaw/journeys/:journey_id/assignments

Parameters

Name	Type	Description	Comments
journey_id	integer	流程记录 id	
查询流程节点某个处理信息
GET /api/v4/yaw/journeys/1/assignments/1295 HTTP/1.1
Authorization: your_authorization
HTTP/1.1 200 OK
{
  "id": 1295,
  "assignee_id": 1,
  "status": "processing",
  "category": "proposed",
  "read": true,
  "operation_data": {
    "operation": "propose",
    "next_vertex_id": 5323
  },
  "current_duration_threshold": null,
  "vertex_id": 5317,
  "journey_id": 830,
  "created_at": "2022-06-20T10:42:48.008+08:00",
  "updated_at": "2022-06-20T10:42:48.109+08:00",
  "response": {
    "id": 2960,
    "created_at": "2021-04-21T11:57:23.102+08:00",
    "cached_values": {
      "8978": {
        "value": [
          {
            "id": 2701,
            "gid": "gid://skylark/Option/2701",
            "value": "新选项2"
          }
        ],
        "text_value": [
          "新选项2"
        ],
        "exported_value": [
          "新选项2"
        ]
      },
      "8979": {
        "value": [
          "2022-06-20"
        ],
        "text_value": [
          "2022-06-20"
        ],
        "exported_value": [
          "2022-06-20"
        ]
      },
      "8980": {
        "value": [
          {
            "id": 9026,
            "gid": "gid://skylark/Attachment/9026",
            "value": "飞书20220304-205019.jpg"
          }
        ],
        "text_value": [
          "飞书20220304-205019.jpg"
        ],
        "exported_value": [
          "飞书20220304-205019.jpg（https://zwfw.cdht.gov.cn/attachments/9026/download）"
        ]
      },
      "11720": {
        "value": [
          "test"
        ],
        "text_value": [
          "test"
        ],
        "exported_value": [
          "test"
        ]
      }
    },
    "mapped_values": {
      "55090a52efe748ed87da86cdaa34d123": {
        "value": [
          {
            "id": 2701,
            "gid": "gid://skylark/Option/2701",
            "value": "新选项2"
          }
        ],
        "text_value": [
          "新选项2"
        ],
        "exported_value": [
          "新选项2"
        ]
      },
      "2157bad17e7b45a5bcc572eac17fa8c0": {
        "value": [
          "2022-06-20"
        ],
        "text_value": [
          "2022-06-20"
        ],
        "exported_value": [
          "2022-06-20"
        ]
      },
      "fa908beece324b31955141147bff6b66": {
        "value": [
          {
            "id": 9026,
            "gid": "gid://skylark/Attachment/9026",
            "value": "飞书20220304-205019.jpg"
          }
        ],
        "text_value": [
          "飞书20220304-205019.jpg"
        ],
        "exported_value": [
          "飞书20220304-205019.jpg（https://zwfw.cdht.gov.cn/attachments/9026/download）"
        ]
      },
      "96987d995bb24e92ab982aa312e8b70a": {
        "value": [
          "test"
        ],
        "text_value": [
          "test"
        ],
        "exported_value": [
          "test"
        ]
      }
    }
  }
},
{
  "id": 1430,
  "assignee_id": 1,
  "status": "processing",
  "category": "processed",
  "read": true,
  "operation_data": {},
  "current_duration_threshold": null,
  "vertex_id": 5323,
  "journey_id": 830,
  "created_at": "2022-06-20T10:42:48.393+08:00",
  "updated_at": "2022-06-20T16:12:15.504+08:00",
  "response": {
    "id": 208895,
    "created_at": "2022-06-20T10:42:48.388+08:00",
    "cached_values": {},
    "mapped_values": {}
  }
}
GET /api/v4/yaw/journeys/:journey_id/assignments/:id

Parameters

Name	Type	Description	Comments
journey_id	integer	流程记录 id
id	integer	节点处理信息 id
修改流程任务状态（通过，回退，转交，撤销）
POST /api/v4/yaw/journeys/:journey_id/assignments/:id  HTTP/1.1
Authorization: your_authorization
{
  "assignment": {
    "response_attributes": {
      "entries_attributes": []
    },
    "comment": "处理意见",
    "operation": "approve",
    "next_vertex_id": 12860,
    "carbon_copy_user_ids": [197],
    "duration_thresholds": [
      { 
        "gid"=>"gid://skylark/YetAnotherWorkflow::Vertex::Normal/7557", 
        "value"=>"2023-07-24 14:29"
      },
      { 
        "gid"=>"gid://skylark/YetAnotherWorkflow::Graph/1079", 
        "value"=>"2023-07-24 14:29"
      }
    ]
  },
  "user_id": "6"
}
PUT /api/v4/yaw/journeys/:journey_id/assignments/:id

Parameters

Name	Type	Description	Comments
journey_id	integer	流程记录 id	
id	integer	任务 id（撤销需要发起者的 assignment id）	
user_id	integer	操作人 id	
assignment[comment]	integer	评论	
assignment[operation]	integer	操作	approve/refuse/refuse/cancel
assignment[next_vertex_id]	integer	下一个节点 id	
assignment[carbon_copy_user_ids]	Array[integer]	抄送者 id	
duration_thresholds[gid]	string	设置时限的 gid	流程时限和节点时限用不同格式区分，格式参考 demo
duration_thresholds[value]	string	处理时限	
流程记录查询
GET /api/v4/yaw/flows/:id/journeys/search HTTP/1.1
Authorization: your_authorization
{
    "query": {
        "8979": {
            "lft": "2022-06-19",
            "rgt": "2022-06-20"
        },
        "-19": "189"
    },
    "page": 1,
    "per_page": 24
}
HTTP/1.1 200 OK
[
    {
        "id": 830,
        "sn": "146920210421115723000002",
        "status": "processing",
        "current_duration_threshold": null,
        "flow_id": 1469,
        "created_at": "2022-06-20T10:42:48.008+08:00",
        "updated_at": "2022-06-20T10:42:48.008+08:00",
        "current_vertex_id": 5323,
        "reviewer_vertex_ids": [
            5317
        ],
        "journey_url": "https://zwfw.cdht.gov.cn/namespaces/1/yet_another_workflow/journeys/830",
        "response": {
            "id": 2959,
            "cached_values": {
                "8978": {
                    "value": [
                        {
                            "id": 2701,
                            "gid": "gid://skylark/Option/2701",
                            "value": "新选项2"
                        }
                    ],
                    "text_value": [
                        "新选项2"
                    ],
                    "exported_value": [
                        "新选项2"
                    ]
                },
                "8979": {
                    "value": [
                        "2022-06-20"
                    ],
                    "text_value": [
                        "2022-06-20"
                    ],
                    "exported_value": [
                        "2022-06-20"
                    ]
                },
                "8980": {
                    "value": [
                        {
                            "id": 9027,
                            "gid": "gid://skylark/Attachment/9027",
                            "value": "飞书20220304-205019.jpg"
                        }
                    ],
                    "text_value": [
                        "飞书20220304-205019.jpg"
                    ],
                    "exported_value": [
                        "飞书20220304-205019.jpg（https://zwfw.cdht.gov.cn/attachments/9027/download）"
                    ]
                },
                "11720": {
                    "value": [
                        "test"
                    ],
                    "text_value": [
                        "test"
                    ],
                    "exported_value": [
                        "test"
                    ]
                }
            },
            "mapped_values": {
                "55090a52efe748ed87da86cdaa34d123": {
                    "value": [
                        {
                            "id": 2701,
                            "gid": "gid://skylark/Option/2701",
                            "value": "新选项2"
                        }
                    ],
                    "text_value": [
                        "新选项2"
                    ],
                    "exported_value": [
                        "新选项2"
                    ]
                },
                "2157bad17e7b45a5bcc572eac17fa8c0": {
                    "value": [
                        "2022-06-20"
                    ],
                    "text_value": [
                        "2022-06-20"
                    ],
                    "exported_value": [
                        "2022-06-20"
                    ]
                },
                "fa908beece324b31955141147bff6b66": {
                    "value": [
                        {
                            "id": 9027,
                            "gid": "gid://skylark/Attachment/9027",
                            "value": "飞书20220304-205019.jpg"
                        }
                    ],
                    "text_value": [
                        "飞书20220304-205019.jpg"
                    ],
                    "exported_value": [
                        "飞书20220304-205019.jpg（https://zwfw.cdht.gov.cn/attachments/9027/download）"
                    ]
                },
                "96987d995bb24e92ab982aa312e8b70a": {
                    "value": [
                        "test"
                    ],
                    "text_value": [
                        "test"
                    ],
                    "exported_value": [
                        "test"
                    ]
                }
            },
            "entries": [
                {
                    "id": 1893464,
                    "field_id": 8979,
                    "option_id": null,
                    "value": "2022-06-20",
                    "choice_id": null,
                    "value_id": null,
                    "latitude": null,
                    "longitude": null,
                    "group_id": null,
                    "detail_id": null
                },
                {
                    "id": 1893463,
                    "field_id": 8980,
                    "option_id": null,
                    "value": "飞书20220304-205019.jpg",
                    "choice_id": null,
                    "value_id": 9026,
                    "latitude": null,
                    "longitude": null,
                    "group_id": null,
                    "detail_id": null,
                    "attachment": {
                        "id": 9026,
                        "name": "飞书20220304-205019.jpg",
                        "size": "469288",
                        "mime_type": "image/jpeg",
                        "extension": "image",
                        "extra_info": {},
                        "download_url": "http://fs-attachment.yqfw.cdyoue.com/1-1655692959-455733ae5fcd10af93ec0ef6f3f46d32-1655692943605?attname=%E9%A3%9E%E4%B9%A620220304-205019.jpg&e=1655729804&token=A02msK5084aaUq-gB1MdVJnPnO9g2p9jD87oMMPg:d32bSp-bEMvZH2jm-1DxxDHIwn4="
                    }
                },
                {
                    "id": 1893462,
                    "field_id": 8978,
                    "option_id": 2701,
                    "value": "新选项2",
                    "choice_id": null,
                    "value_id": null,
                    "latitude": null,
                    "longitude": null,
                    "group_id": null,
                    "detail_id": null
                },
                {
                    "id": 1893461,
                    "field_id": 11720,
                    "option_id": null,
                    "value": "test",
                    "choice_id": null,
                    "value_id": null,
                    "latitude": null,
                    "longitude": null,
                    "group_id": null,
                    "detail_id": null
                }
            ]
        },
        "user": {
            "id": 1,
            "name": "高新服务小助手",
            "nickname": "坤",
            "sex": 0,
            "phone": "180000000",
            "identifier": "180000000",
            "openid": "oyyT9s3lmVAnPV9CwYHL2zFLMf4E",
            "created_at": "2020-07-13T16:27:44.749+08:00",
            "updated_at": "2022-05-22T21:10:16.405+08:00",
            "headimgurl": "http://thirdwx.qlogo.cn/mmopen/vi_32/Ok2zIF8CS9AP28EWv75Ox9uVahUVtWN21ic3IxMILCpudNUl59Gcia94WtqU8aLz4t5PIpT9F84sYIMBr5oGDLGw/96"
        }
    }
]
GET /api/v4/yaw/flows/:id/journeys/search

Parameters

Name	Type	Description	Comments
id	integer	流程 id	
获取某个用户发起的流程
GET /api/v4/yaw/flows/:flow_id/journeys/proposed_journeys HTTP/1.1
Authorization: your_authorization
HTTP/1.1 200 OK
[
  {
      "id": 7802,
      "status": "processing",
      "category": "proposed",
      "read": true,
      "current_duration_threshold": null,
      "created_at": "2020-11-18T16:05:08.207+08:00",
      "updated_at": "2020-11-18T16:05:08.261+08:00",
      "after_submit_action": "default",
      "after_submit_redirect_url": null,
      "gid": "gid://skylark/YetAnotherWorkflow::Assignment/7802",
      "pretty_current_duration_threshold": null,
      "journey": {
          "id": 2236,
          "sn": "145920201118152008000001",
          "custom_sn": null,
          "status": "processing",
          "current_duration_threshold": null,
          "created_at": "2020-11-18T16:05:08.207+08:00",
          "updated_at": "2020-11-18T16:05:08.207+08:00",
          "gid": "gid://skylark/YetAnotherWorkflow::Journey/2236",
          "pretty_current_duration_threshold": null,
          "response": {
              "id": 78149,
              "created_at": "2020-11-18T15:20:08.063+08:00",
              "entries": []
          },
          "user": {
              "id": 79,
              "name": "樊翔宇",
              "nickname": "k k",
              "phone": "180000000",
              "identifier": "180000000",
              "qq": null,
              "headimgurl": "https://thirdwx.qlogo.cn/mmopen/vi_32/Q0j4TwGTfTKFLWWPT1sSVywib8qpNNfLjMOliblYqa105ibCOKGvzwRV0vAEAlGLcwXia8AK9FQiavG1tDxcCCkfUCA",
              "openid": "oXDm5s2v7fnJ2Mut-UiiHtCEIb6Q",
              "imported_alias": "樊翔宇",
              "gid": "gid://skylark/User/79"
          }
      },
      "response": {
          "id": 78150,
          "created_at": "2020-11-18T15:20:08.097+08:00",
          "entries": [
              {
                  "id": 1141143,
                  "field_id": 7202,
                  "option_id": null,
                  "value": "WechatIMG132.jpeg",
                  "choice_id": null,
                  "value_id": 3839,
                  "latitude": null,
                  "longitude": null,
                  "group_id": null,
                  "detail_id": null,
                  "attachment": {
                      "id": 3839,
                      "name": "WechatIMG132.jpeg",
                      "size": "16998",
                      "mime_type": "image/jpeg",
                      "extension": "image",
                      "extra_info": {}
                  }
              },
              {
                  "id": 1141142,
                  "field_id": 7201,
                  "option_id": null,
                  "value": "2020-11-18",
                  "choice_id": null,
                  "value_id": null,
                  "latitude": null,
                  "longitude": null,
                  "group_id": null,
                  "detail_id": null
              },
              {
                  "id": 1141141,
                  "field_id": 7200,
                  "option_id": 8854,
                  "value": "新选项",
                  "choice_id": null,
                  "value_id": null,
                  "latitude": null,
                  "longitude": null,
                  "group_id": null,
                  "detail_id": null
              },
              {
                  "id": 1141140,
                  "field_id": 7199,
                  "option_id": null,
                  "value": "test",
                  "choice_id": null,
                  "value_id": null,
                  "latitude": null,
                  "longitude": null,
                  "group_id": null,
                  "detail_id": null
              }
          ]
      }
  }
]
GET /api/v4/yaw/flows/:flow_id/journeys/proposed_journeys

Parameters

Name	Type	Description	Comments
flow_id	integer	流程 id	
user_id	integer	用户 id	
获取某个用户处理的流程任务列表
GET /api/v4/yaw/flows/user_assignments.json?user_id=:user_id&category=:category HTTP/1.1
Authorization: your_authorization
HTTP/1.1 200 OK
[
    {
        "id": 4057665,
        "assignee_id": 6,
        "status": "processing",
        "category": "proposed",
        "read": true,
        "operation_data": {
            "operation": "propose",
            "next_vertex_id": 30273,
            "duration_thresholds": {
                "gid": "gid://skylark/YetAnotherWorkflow::Graph/2899",
                "value": "2022-08-13 22:55"
            }
        },
        "current_duration_threshold": null,
        "vertex_id": 30229,
        "journey_id": 442855,
        "created_at": "2022-07-27T19:35:36.045+08:00",
        "updated_at": "2022-07-27T19:35:36.203+08:00",
        "response": {
            "id": 5083740,
            "created_at": "2022-07-27T19:35:35.606+08:00",
            "cached_values": {
                "20026": {
                    "value": [
                        {
                            "id": 43664,
                            "gid": "gid://skylark/Option/43664",
                            "value": "综合办公室"
                        }
                    ],
                    "text_value": [
                        "综合办公室"
                    ],
                    "exported_value": [
                        "综合办公室"
                    ]
                }
            },
            "mapped_values": {
                "department": {
                    "value": [
                        {
                            "id": 43664,
                            "gid": "gid://skylark/Option/43664",
                            "value": "综合办公室"
                        }
                    ],
                    "text_value": [
                        "综合办公室"
                    ],
                    "exported_value": [
                        "综合办公室"
                    ]
                }
            }
        }
    },
    ...
]
GET /api/v4/yaw/flows/user_assignments.json?user_id=:user_id&category=:category

Parameters

Name	Type	Description	Comments
user_id	integer	用户 id	
category	integer	任务类型	category 固定为: proposed「我发起的」，processed 「由我处理」，cc「抄送我的」
流程编号搜索流程任务记录
GET /api/v4/yaw/flows/1/journeys/find_by_sn?sn=6520230224115606001030 HTTP/1.1
Authorization: your_authorization
HTTP/1.1 200 OK
{
    "id": 830,
    "sn": "6520230224115606001030",
    "status": "processing",
    "current_duration_threshold": null,
    "flow_id": 1469,
    "created_at": "2022-06-20T10:42:48.008+08:00",
    "updated_at": "2022-06-20T10:42:48.008+08:00",
    "current_vertex_id": 5323,
    "reviewer_vertex_ids": [
        5317
    ],
    "journey_url": "https://zwfw.cdht.gov.cn/namespaces/1/yet_another_workflow/journeys/830",
    "response": {}
}
GET //api/v4/yaw/flows/:flow_id/journeys/find_by_sn

Parameters

Name	Type	Description	Comments
sn	string	流程编号	
flow_id	integer	流程 id	
获取当前流程任务的处理者
GET /api/v4/yaw/flows/1/journeys/1/current_processing_users HTTP/1.1
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
    "tags": []
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
    "tags": []
  }
]
GET /api/v4/yaw/flows/:flow_id/journeys/:id/current_processing_users

Parameters

Name	Type	Description	Comments
flow_id	integer	流程 id	
id	integer	流程任务 id
查询流程节点的详细信息
GET /api/v4/yaw/flows/:flow_id/vertices/:id HTTP/1.1
Authorization: your_authorization
HTTP/1.1 200 OK
{
  "id":113,
  "name":"开始节点",
  "type":"YetAnotherWorkflow::Vertex::Initial",
  "alias_name": "",
  "user_boundary": {},
  "fields": []
}
GET /api/v4/yaw/flows/:flow_id/vertices/:id

Parameters

Name	Type	Description	Comments
flow_id	integer	流程 id	
id	integer	节点 id	
终止流程任务
PUT /api/v4/yaw/flows/:flow_id/journeys/:id HTTP/1.1
Authorization: your_authorization

{
 "status": "aborted"
}
HTTP/1.1 200 OK
{
  "id":113,
  "status": "aborted"
}
PUT /api/v4/yaw/flows/:flow_id/journeys/:id

Parameters

Name	Type	Description	Comments
flow_id	integer	流程 id
id	integer	流程任务 id