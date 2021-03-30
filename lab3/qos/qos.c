#include "rte_common.h"
#include "rte_mbuf.h"
#include "rte_meter.h"
#include "rte_red.h"

#include "qos.h"

#define COLOR_NUM 3

// meter
struct rte_meter_srtcm_params app_srtcm_params = {
	.cir = 1000000 * 46,
	.cbs = 2048,
	.ebs = 2048
};
struct rte_meter_srtcm_profile app_srtcm_profile;

struct rte_meter_srtcm app_flows[APP_FLOWS_MAX];

//red
struct rte_red_config red_params[COLOR_NUM];
struct rte_red red_datas[APP_FLOWS_MAX][COLOR_NUM];
unsigned red_queues[APP_FLOWS_MAX][COLOR_NUM] = {};

/**
 * srTCM
 */
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

enum qos_color
qos_meter_run(uint32_t flow_id, uint32_t pkt_len, uint64_t time)
{
    /* to do */
    return rte_meter_srtcm_color_blind_check(&app_flows[flow_id], &app_srtcm_profile, time, pkt_len);
}


/**
 * WRED
 */

int
qos_dropper_init(void)
{
    /* to do */
    int ret;
    enum qos_color color;
    for (color = GREEN; color <= RED; color++)
    {
        if (color != RED)
            ret = rte_red_config_init(&red_params[color], 9, 1022, 1023, 10);
        else 
            ret = rte_red_config_init(&red_params[color], 9, 0, 1, 10);

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

int
qos_dropper_run(uint32_t flow_id, enum qos_color color, uint64_t time)
{
    /* to do */
    static uint64_t latest_time = 0;

    if(time != latest_time)
    {
        memset(red_queues, 0, sizeof(red_queues));
        for (int i = 0; i < APP_FLOWS_MAX; i++)
            for (int j = 0; j < COLOR_NUM; j++)
                rte_red_mark_queue_empty(&red_datas[i][j], time);
    } 

    latest_time = time;

    int result = rte_red_enqueue(&red_params[color], &red_datas[flow_id][color], red_queues[flow_id][color], time);
    if (!result)
        red_queues[flow_id][color]++;

    return result;
}