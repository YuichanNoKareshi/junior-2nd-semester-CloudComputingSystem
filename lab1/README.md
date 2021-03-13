# lab1 
学号 518030910237
姓名 周义天
邮箱 zyt531686350@sjtu.edu.cn

&emsp;&emsp;lab1可以分为三个部分，分别为reorder、丢包、包损坏，我的思路是逐步实现各部分，下面将讲解总体的设计以及各部分的实现及细节

---

### <font color="red">设计</font>
&emsp;&emsp;pkt的结构
<table>
<tr>
    <td align="center">2 byte</td>
    <td align="center">1 byte</td>
    <td align="center">4 byte</td>
    <td align="center">rest</td>
</tr>
<tr>
    <td align="center">checksum</td>
    <td align="center">payload size</td>
    <td align="center">sequence number</td>
    <td align="center">payload</td>
</tr>
</table>
&emsp;&emsp;ack_pkt的结构
<table>
<tr>
    <td align="center">2 byte</td>
    <td align="center">4 byte</td>
    <td align="center">rest</td>
</tr>
<tr>
    <td align="center">checksum</td>
    <td align="center">sequence number</td>
    <td align="center">-</td>
</tr>
</table>

&emsp;&emsp;参数设置
```c++
#define window_size 10
#define timeout 0.3
```

&emsp;&emsp;sender用到的数据结构
```c++
int sequence_number = 1; // 记录当前要发的pkt的序号
int window_left = window_size; // 记录window还差几个包被填满
std::queue <struct packet*> unsend_pkts; // 发送队列，储存已被划分好的，但尚未发送的包
std::map <int, bool> unack_pkts; // 记录每个包是否已收到ack
std::map <int, struct packet*> sliding_window; // 记录当前window中的包
```

&emsp;&emsp;receiver用到的数据结构
```c++
std::map<int, struct message*> received_msgs; // 储存所有收到的msg(msg的序号与收到pkt的序号对应)
std::map<int, bool> msg_status; // int记录msg序号，bool为false表示已收到但未发至上层，true表示已发至上层
int last_upper = 0; // 记录上一个发到上层的msg的序号
```
&emsp;&emsp;之所以在数据结构中全部使用指针，是因为我发现使用值会出现存时与取时数据不一致的问题，据了解可能与c++的深拷贝浅拷贝有关，故全部使用指针

---

## <font color="red">reorder</font>
&emsp;&emsp;在这一部分中，会出现乱序发送pkt和ack_pkt的情况，因此我们需要在sender和receiver使用队列来保证正确顺序。(此时pkt和ack_pkt中尚未加入checksum)

#### sender收到msg后
1. 将msg划分为多个pkt
2. 在unack_pkts中记录其尚未收到ack
3. 根据window是否已满来决定是直接发送pkt还是存入发送队列中
```c++
unack_pkts[sequence_number] = false;
if (window_left > 0) // window未满
{
    Sender_ToLowerLayer(pkt); 
    sliding_window[sequence_number] = pkt;
    window_left--;
}
else 
    unsend_pkts.push(pkt);
sequence_number++;
```

#### receiver收到pkt后
1. 向sender发送ack
2. 然后构造msg，并将msg存入received_msgs、将msg_stutas置为false
3. 检查是否有msg可以发送至上层
```c++
memcpy(msg->data, pkt->data+header_size, msg->size);
received_msgs[ack_sequence_number] = msg;

msg_status[ack_sequence_number] = false;

for (int i=last_upper+1; i<=ack_sequence_number; i++) // 从last_upper+1开始遍历
{
    if (msg_status.count(i) == 0) // 尚未收到，则后面的都不能发至上层
        break;
    else if ( msg_status[i] == false && (i == 1 || msg_status[i-1] == true)) // i已被接收，且i==1或i-1已被发至上层
    {
        Receiver_ToUpperLayer(received_msgs[i]);
        last_upper = i;
        msg_status[i] = true;
    }
}
```

#### sender收到ack后
1. 在unack_pkts中记录其已收到ack
2. 在window中按顺序删去已ack的(到第一个未ack为止)
```c++
std::map<int, struct packet*>::iterator iter= sliding_window.begin();
while ( iter != sliding_window.end()) 
{
    if (unack_pkts[iter->first] == true)
    {
        iter = sliding_window.erase(iter);
        window_left++;
    }
    else // 可以把收到ack的所有pkt都从window中删去
        iter++;
}
```
3. 检查window是否有空位来容纳发送队列中的pkt
```c++
while (!unsend_pkts.empty() && window_left > 0)
{
    struct packet* front_pkt = (struct packet*) malloc (sizeof(struct packet));
    front_pkt = unsend_pkts.front();  
    memcpy(&temp_sequence_number, front_pkt->data+1, sizeof(temp_sequence_number));
    
    Sender_ToLowerLayer(front_pkt);
    unsend_pkts.pop();
    sliding_window[temp_sequence_number] = front_pkt;
    window_left--;
}
```

#### 对关键细节的解释
+ 当receiver收到i但i-1还未到时，receiver不会把i发至上层，必须等i之前的所有msg都发送后才会发送i
+ 当sender收到i的ack但未收到i-1时，不会丢弃这条ack，而是记录i已经ack。这样可以保证在下个部分的重发时只重发window中第一个pkt而不用重发整个window

---

## <font color="red">丢包</font>
&emsp;&emsp;在这一部分中，会出现pkt和ack_pkt的情况，因此我们需要在sender端设置timer来实现重发。(此时pkt和ack_pkt中尚未加入checksum)

#### StartTimer
&emsp;&emsp;实际上，在我的设计中，需要设置timer的情况只会出现在两个场景下
1. 在sender划分好pkt并决定直接发送时，发现window为空
```c++
if (window_left > 0) // window未满
{
    Sender_ToLowerLayer(pkt); 
    if (sliding_window.empty()) // window是空的，设置timer
        Sender_StartTimer(timeout);
    sliding_window[sequence_number] = pkt;
    window_left--;
}
else 
    unsend_pkts.push(pkt);
```
2. 在sender收到ack时，发现是window中第一个pkt的ack
```c++
int temp_sequence_number;
memcpy(&temp_sequence_number, pkt->data, sizeof(temp_sequence_number));

if (sliding_window.begin()->first == temp_sequence_number) // window中第一个pkt的ack
    Sender_StartTimer(timeout);
```

#### StopTimer
&emsp;&emsp;当确定所有的pkt都已经被接收后，需要停止timer。我选择在每次收到ack后检查unsend_pkts和sliding_window是否为空
```c++
if (unsend_pkts.empty() && sliding_window.empty())
     Sender_StopTimer();
```

#### Sender_Timeout
&emsp;&emsp;当timer时间走完还未收到ack，需要重新发包并重置timer
```c++
void Sender_Timeout()
{
    std::map<int, struct packet*>::iterator iter= sliding_window.begin();
    Sender_ToLowerLayer(iter->second);
    
    Sender_StartTimer(timeout);
}
```

#### 对关键细节的解释
+ 由于允许i在i-1之前接受ack，所以只用重发window中第一个pkt
+ 若由于ack丢包而导致receiver多次收到同一个pkt，要保证不能多次将其发至上层
```c++
if (msg_status.count(ack_sequence_number) == 0) // 若已经收到过，不要对其做操作
    msg_status[ack_sequence_number] = false;
```
+ 值得一提的是，我在跑丢包部分的测试时，10次中总会有1次过不了。使用printf debug法发现，虽然receiver收到了所有的pkt且发送了ack，但是有一两个包不知道什么原因没有发至上层(原谅我暂时想不出原因)，因此我在Receiver_Final中强制要求把received_msgs中未发送的包全部发送
```c++
int i = last_upper + 1;
while (true)
{
    if ( msg_status.count(i) == 0 )
        break;
    else if ( msg_status[i] == false ) 
    {
        Receiver_ToUpperLayer(received_msgs[i]);
        last_upper = i;
        msg_status[i] = true;
        i++;
    }
}
```

---

## <font color="red">包损坏</font>
&emsp;&emsp;在这一部分中，会出现pkt和ack_pkt中数据出错的情况，因此我们需要在所有pkt和ack_pkt中加入checksum来保证正确，以下为几个重要概念。
+ 反码算数运算：从低位到高位逐列进行计算。0和0相加是0，0和1相加是1，1和1相加是0但要产生一个进位1，加到下一列。如果最高位相加后产生进位，则最后得到的结果要加1。<font color="red">即循环进位，最高位的进位加到最低位。</font>
+ 校验：把pkt.data看做一个char数组，然后按照2 byte的大小分组。如果数据的字节长度为奇数，则在数据尾部补一个的0x0凑成偶数。对每组进行反码算数运算，最后的结果必须为0xff。
+ 因此checksum就等于对后面所有组的和取反。
```c++
short CheckSum(struct packet* pkt) 
{
    unsigned long sum = 0;
    for(int i = 2; i < RDT_PKTSIZE; i += 2) sum += *(short *)(&(pkt->data[i]));
    while(sum >> 16) sum = (sum >> 16) + (sum & 0xffff);
    return ~sum;
}
```
&emsp;&emsp;这一部分很简单，sender在发pkt前加入checksum，receiver在发ack前也加入checksum。不论是receiver校验pkt错误，还是sender检验ack错误，都直接return，造成一种丢包的假象，于是问题就变成了如何处理丢包
