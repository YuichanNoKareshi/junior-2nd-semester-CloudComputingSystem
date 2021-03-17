#include <stdint.h>
#include <inttypes.h>
#include <rte_eal.h>
#include <rte_ethdev.h>
#include <rte_cycles.h>
#include <rte_lcore.h>
#include <rte_mbuf.h>

#include <rte_ether.h>
#include <rte_udp.h>
#include <rte_ip.h>

#define RX_RING_SIZE 1024
#define TX_RING_SIZE 1024

#define NUM_MBUFS 8191
#define MBUF_CACHE_SIZE 250
#define BURST_SIZE 32

static const struct rte_eth_conf port_conf_default = {
	.rxmode = {
		.max_rx_pkt_len = RTE_ETHER_MAX_LEN,
	},
};

/*
 * Initializes a given port using global settings and with the RX buffers
 * coming from the mbuf_pool passed as a parameter.
 */
static inline int
port_init(struct rte_mempool *mbuf_pool)
{
	struct rte_eth_conf port_conf = port_conf_default;
	const uint16_t rx_rings = 1, tx_rings = 1;
	int retval;
	uint16_t q;

	retval = rte_eth_dev_configure(0, rx_rings, tx_rings, &port_conf);
	if (retval != 0) 
		return retval;

	/* Allocate and set up 1 RX queue per Ethernet port. */
	for (q = 0; q < rx_rings; q++) {
		retval = rte_eth_rx_queue_setup(0, q, RX_RING_SIZE,
				rte_eth_dev_socket_id(0), NULL, mbuf_pool);
		if (retval < 0)
			return retval;
	}

	/* Allocate and set up 1 TX queue per Ethernet port. */
	for (q = 0; q < tx_rings; q++) {
		retval = rte_eth_tx_queue_setup(0, q, RX_RING_SIZE,
				rte_eth_dev_socket_id(0), NULL);
		if (retval < 0)
			return retval;
	}

	/* Start the Ethernet port. */
	retval = rte_eth_dev_start(0);
	if (retval < 0)
		return retval;

	return 0;
}

/*
 * The main function, which does initialization and calls the per-lcore
 * functions.
 */
int
main(int argc, char *argv[])
{
	struct rte_mempool *mbuf_pool;

	/* Initialize the Environment Abstraction Layer (EAL). */
	int ret = rte_eal_init(argc, argv);
	if (ret < 0)
		rte_exit(EXIT_FAILURE, "Error with EAL initialization\n");

	argc -= ret;
	argv += ret;

	/* Creates a new mempool in memory to hold the mbufs. */
	mbuf_pool = rte_pktmbuf_pool_create("MBUF_POOL", NUM_MBUFS,
		MBUF_CACHE_SIZE, 0, RTE_MBUF_DEFAULT_BUF_SIZE, rte_socket_id());

	if (mbuf_pool == NULL)
		rte_exit(EXIT_FAILURE, "Cannot create mbuf pool\n");

	/* Initialize all ports. */
    if (port_init(mbuf_pool) != 0)
        rte_exit(EXIT_FAILURE, "Cannot init port %"PRIu16 "\n",0);

    struct rte_mbuf* tx_pkts[BURST_SIZE]; //buffer of packets 
    for (int i=0; i<BURST_SIZE; i++)
    {   
        tx_pkts[i] = rte_pktmbuf_alloc(mbuf_pool);
        
        struct rte_ether_hdr* eth_hdr = rte_pktmbuf_mtod(tx_pkts[i], struct rte_ether_hdr*);
        struct rte_ipv4_hdr* ip_hdr = (struct rte_ipv4_hdr*)(rte_pktmbuf_mtod(tx_pkts[i], char*) 
                                        + sizeof(struct rte_ether_hdr));
        struct rte_udp_hdr* udp_hdr = (struct rte_udp_hdr*)(rte_pktmbuf_mtod(tx_pkts[i], char*) 
                                        + sizeof(struct rte_ether_hdr) + sizeof(struct rte_ipv4_hdr));
        int* data = (int*)(rte_pktmbuf_mtod(tx_pkts[i], char*) + sizeof(struct rte_ether_hdr) 
                                        + sizeof(struct rte_ipv4_hdr) + sizeof(struct rte_udp_hdr));

        struct rte_ether_addr d_addr = {{0x14, 0x02, 0xEC, 0x89, 0x8D, 0x24}};
        struct rte_ether_addr s_addr = {{0x14, 0x02, 0xEC, 0x89, 0xED, 0x54}};

        eth_hdr->d_addr = d_addr;                                   /**< Destination address. */
        eth_hdr->s_addr = s_addr;                                   /**< Source address. */
        eth_hdr->ether_type = 0x0008;                               /**< Frame type. */

        ip_hdr->version_ihl = RTE_IPV4_VHL_DEF;                     /**< version and header length */
        ip_hdr->type_of_service = RTE_IPV4_HDR_DSCP_MASK;           /**< type of service */
        ip_hdr->total_length = 0x2000;                              /**< length of packet */
        ip_hdr->packet_id = 0;                                      /**< packet ID */
        ip_hdr->fragment_offset = 0;                                /**< fragmentation offset */
        ip_hdr->time_to_live = 100;                                 /**< time to live */
        ip_hdr->next_proto_id = 17;                                 /**< protocol ID */
        ip_hdr->hdr_checksum = rte_ipv4_cksum(ip_hdr);              /**< header checksum */
        ip_hdr->src_addr = 0;                                       /**< source address */
        ip_hdr->dst_addr = 0;                                       /**< destination address */

        udp_hdr->src_port = 80;                                     /**< UDP source port. */
        udp_hdr->dst_port = 8080;                                   /**< UDP destination port. */
        udp_hdr->dgram_len = 0x0c00;                                /**< UDP datagram length */
        udp_hdr->dgram_cksum = 1;                                   /**< UDP datagram checksum */

        *data = i;
        tx_pkts[i]->data_len = sizeof(struct rte_ether_hdr) + sizeof(struct rte_ipv4_hdr) 
                            + sizeof(struct rte_udp_hdr) + sizeof(int);
        tx_pkts[i]->pkt_len = sizeof(struct rte_ether_hdr) + sizeof(struct rte_ipv4_hdr) 
                            + sizeof(struct rte_udp_hdr) + sizeof(int);
    }
        
    uint16_t nb_tx = rte_eth_tx_burst(0, 0, tx_pkts, BURST_SIZE);
    printf("successfully sent %d packets\n", nb_tx);

    for (int i=0; i<BURST_SIZE; i++)
        rte_pktmbuf_free(tx_pkts[i]);

	return 0;
}
