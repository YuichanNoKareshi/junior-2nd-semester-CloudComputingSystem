# qos lab
## Task one: implement a meter and a dropper
&emsp;&emsp;qos_meter_init和qos_meter_run主要参考DPDK/examples/qos_meter/main.c
#### 全局变量

```c++
struct rte_meter_srtcm_params app_srtcm_params = {
	.cir = 1000000 ,
	.cbs = 1024,
	.ebs = 1024
};
struct rte_meter_srtcm_profile app_srtcm_profile;
struct rte_meter_srtcm app_flows[APP_FLOWS_MAX];

struct rte_red_config red_params[COLOR_NUM];
struct rte_red red_datas[APP_FLOWS_MAX][COLOR_NUM];
unsigned red_queues[APP_FLOWS_MAX][COLOR_NUM] = {};
uint64_t latest_time = 0;
```
+ meter
  + app_srtcm_params用来提供配置srtcm的相关参数
  + app_flows用来记录各个流的参数
  + app_srtcm_profile是params用来给flows配置参数的中间配置文件
+ dropper
  + red_params记录red的配置参数
  + red_datas记录red的run-time data
  + red_queues记录每个流每个颜色最近drop的次数
  + latest_time记录上一次time

#### qos_meter_init：执行meter的初始化逻辑
```c++
int
qos_meter_init(void)
{
    uint32_t i;
	int ret;

	ret = rte_meter_srtcm_profile_config(&app_srtcm_profile,
		&app_srtcm_params);
	if (ret)
		return ret;

	for (i = 0; i < APP_FLOWS_MAX; i++) {
		ret = rte_meter_srtcm_config(&app_flows[i], &app_srtcm_profile);
		if (ret)
			return ret;
	}

	return 0;
}
```

#### qos_meter_run：由pkt_len来推测当前包的类别

```c++
enum qos_color
qos_meter_run(uint32_t flow_id, uint32_t pkt_len, uint64_t time)
{
    /* to do */
    return rte_meter_srtcm_color_blind_check(&app_flows[flow_id], &app_srtcm_profile, time, pkt_len);
}
```

#### qos_dropper_init：初始化red_params和red_datas
```c++
int
qos_dropper_init(void)
{
    /* to do */
    int ret;
    enum qos_color color;
    for (color = GREEN; color <= RED; color++)
    {
        if (color != RED)
            ret = rte_red_config_init(&red_params[color], 1, 1022, 1023, 10);
        else 
            ret = rte_red_config_init(&red_params[color], 1, 0, 1, 10);

        if (ret)
            return ret;

        for (int i=0; i < APP_FLOWS_MAX; i++)
        {   
            if (rte_red_rt_data_init(&red_datas[i][color]) != 0)
                rte_panic("Cannot init RED data.\n");
        }
    }

    return 0;
}
```

#### qos_dropper_run：对包执行drop
```c++
int
qos_dropper_run(uint32_t flow_id, enum qos_color color, uint64_t time)
{
    if(time != latest_time)
    {
        memset(red_queues, 0, sizeof(red_queues));
        
        for (int i = 0; i < APP_FLOWS_MAX; i++)
            for (int j = 0; j < COLOR_NUM; j++)
                rte_red_mark_queue_empty(&red_datas[i][j], time);

        latest_time = time;
    } 
    
    int result = rte_red_enqueue(&red_params[color], &red_datas[flow_id][color], red_queues[flow_id][color], time);
    if (!result)
        red_queues[flow_id][color]++;

    return result;
}
```
+ 我们根据flow_id对包进行分类，使用不同的配置参数来执行drop逻辑
+ time由main函数进行管理，time每次改变，都需要清除所有包
+ 丢包通过rte_red_enqueue实现，返回值为0表示不drop，否则drop
---

## Task two: deduce parameters
1. 初始数据
```c++
if (color != RED)
    ret = rte_red_config_init(&red_params[color], 1, 1022, 1023, 10);
else 
    ret = rte_red_config_init(&red_params[color], 1, 0, 1, 10);
```
flow|cir|cbs|ebs|pass
-|-|-|-|-
0|1000000|1024|1024|14354
1|1000000|1024|1024|15615
2|1000000|1024|1024|14358
3|1000000|1024|1024|15156

2. 先使 flow0 的 pass 达到 1.28G (= 160M/s)左右，通过一系列调整基本达到

flow|cir|cbs|ebs|pass
-|-|-|-|-
0|16000000|160000|1600000|1687657
1|16000000|160000|1600000|1690229
2|16000000|160000|1600000|1623607
3|16000000|160000|1600000|1631986

3. 再使各个流的pass达到8:4:2:1，但是无论怎么改参数都，都是1:1:1:1，向同学咨询后发现是自己只用了一个app_srtcm_profile，在qos_meter_run中无论flow_id为多少都只会使用这个，于是我修改了app_srtcm_profile的个数
```c++
enum qos_color
qos_meter_run(uint32_t flow_id, uint32_t pkt_len, uint64_t time)
{
    /* to do */
    return rte_meter_srtcm_color_blind_check(&app_flows[flow_id], &app_srtcm_profile[flow_id], time, pkt_len);
}
```
flow|cir|cbs|ebs|pass
-|-|-|-|-
0|16000000|160000|1600000|1602549
1|8000000|80000|800000|912628
2|4000000|40000|400000|462481
3|2000000|20000|200000|235271

4. 继续使pass接近比值，我的想法是每个流的cir、cbs、ebs都是上一个的1/2

flow|cir|cbs|ebs|pass
-|-|-|-|-
0|16000000|80000|1400000|1539111
1|8000000|40000|700000|771985
2|4000000|20000|350000|391927
3|2000000|10000|175000|198949

flow|cir|cbs|ebs|pass
-|-|-|-|-
0|16000000|60000|1500000|1619294
1|8000000|30000|750000|813519
2|4000000|15000|375000|409996
3|2000000|7500|187500|209666

flow|cir|cbs|ebs|pass
-|-|-|-|-
0|16000000|60000|1450000|1568680
1|8000000|30000|725000|790730
2|4000000|15000|362500|399226
3|2000000|7500|181250|203583

flow|cir|cbs|ebs|pass
-|-|-|-|-
0|16000000|70000|1450000|1578753
1|8000000|35000|725000|792624
2|4000000|17500|362500|400986
3|2000000|8750|181250|203510

flow|cir|cbs|ebs|pass
-|-|-|-|-
0|16000000|100000|1450000|1609656
1|8000000|50000|725000|810363
2|4000000|25000|362500|409547
3|2000000|12500|181250|208835

flow|cir|cbs|ebs|pass
-|-|-|-|-
0|16000000|50000|1500000|1608016
1|8000000|25000|750000|810385
2|4000000|12500|375000|409469
3|2000000|6250|187500|209565

&emsp;&emsp;于是我决定以最后一组数据作为我的结论...
