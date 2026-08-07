import React, {useState, useEffect} from 'react';
import {
    List, Spin, Layout, Typography,
    Badge, Modal, Input, Button, message
} from 'antd';
import {
    WifiOutlined, SyncOutlined, ClockCircleOutlined, ManOutlined, SettingOutlined, LinkOutlined
} from '@ant-design/icons';
import {GetOnlinePeers, ConnectTo, SetClientInfo} from "../bindings/virtualnet/kvirnet/clientservice.ts";
import {Events} from "@wailsio/runtime";


const {Header, Content, Footer} = Layout;
const {Title, Text} = Typography;


interface InfoSettingModalProps {
    open: boolean
    onClose: () => void
}


const InfoSettingModal = ({onClose, open}: InfoSettingModalProps) => {
    const [isModalOpen, setIsModalOpen] = useState(open);
    const [name, setName] = useState('')

    useEffect(() => {
        setIsModalOpen(open)
    }, [open]);

    const handleOk = async () => {
        setIsModalOpen(false);

        await SetClientInfo(name)
        onClose()
    };


    const handleCancel = () => {
        setIsModalOpen(false);
        onClose()

    };

    return (
        <>
            <Modal
                title="设置信息"
                closable={{'aria-label': 'Custom Close Button'}}
                open={isModalOpen}
                onOk={handleOk}
                okText="确定"
                cancelText="取消"
                onCancel={handleCancel}

            >
                <Input placeholder="名字" onChange={(e) => setName(e.target.value)}/>
            </Modal>
        </>
    );
};

// VPN节点数据类型定义
interface VpnNode {
    id: string;
    ip: string;
    latency: number; // 延迟(ms)
    connected: boolean; // 是否已建立连接
    isLocal: boolean; // 是否为本机
}

const App: React.FC = () => {
    // 状态管理
    const [nodes, setNodes] = useState<VpnNode[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [openSetting, setOpenSetting] = useState(true)
    const [connecting, setConnecting] = useState<string | null>(null)


    // 加载在线节点列表
    useEffect(() => {
        const fetchData = async () => {

            const data = await GetOnlinePeers()

            setNodes(data
                .filter(v => v != null)
                .map(v => {
                return {
                    id: v.name,
                    ip: v.virtualIp,
                    latency: v.latency,
                    connected: v.connected,
                    isLocal: v.isLocal
                }
            }));
            setLoading(false);
        };

        fetchData();

        const onClientChange = () => {
            fetchData()
        }

        Events.On('clientChanged', onClientChange)
        return () => {
            Events.Off("clientChanged")
        }
    }, []);


    // 延迟显示样式
    const getLatencyDisplay = (latency: number) => {

        if (latency === -1) {
            return <Spin size="small" children={<SyncOutlined spin/>}/>;
        }

        let color = 'green';
        if (latency > 100) color = 'orange';
        if (latency > 200) color = 'red';

        return (
            <Text style={{color}}>
                {latency}ms <ClockCircleOutlined/>
            </Text>
        );
    };

    // 点击连接
    const handleConnect = async (ip: string) => {
        setConnecting(ip)
        try {
            await ConnectTo(ip)
            message.success('连接请求已发送')
        } catch (e) {
            message.error(String(e))
        } finally {
            setConnecting(null)
        }
    };

    // 右侧操作区：根据状态显示本机标签/连接按钮/已连接标签
    const getActionDisplay = (item: VpnNode) => {
        if (item.isLocal) {
            return <Badge status="processing" text="本机"/>
        }
        if (item.connected) {
            return <Badge status="success" text="已连接"/>
        }
        return (
            <Button
                type="primary"
                size="small"
                icon={<LinkOutlined/>}
                loading={connecting === item.ip}
                onClick={() => handleConnect(item.ip)}
            >
                连接
            </Button>
        )
    };


    return (
        <Layout style={{
            display: 'flex',
            flexDirection: 'column',
            minHeight: '100vh',
            width: '100%',
            margin: 0,
            padding: 0
        }}>
            {openSetting && <InfoSettingModal open={openSetting} onClose={() => {
                setOpenSetting(false)
            }}></InfoSettingModal>}

            <Header style={{background: '#fff', padding: '0 20px', boxShadow: '0 2px 8px rgba(0,0,0,0.1)'}}>
                <div style={{display: 'flex', alignItems: 'center', height: '100%'}}>
                    <WifiOutlined style={{fontSize: '24px', color: '#1890ff', marginRight: '12px'}}/>
                    <Title level={3} style={{margin: 0}}>虚拟局域网</Title>
                    <div style={{flexGrow: 1}}></div>
                    <Button shape="circle" icon={<SettingOutlined />} onClick={() => setOpenSetting(true)}></Button>
                </div>
            </Header>

            <Content style={{
                padding: '24px',
                height: '75vh',
                overflowY: 'auto',
                maxWidth: '1200px',
                margin: '0 auto',
                width: '100%'
            }}>


                <Spin style={{width: '100%'}} spinning={loading}>
                    <List
                        itemLayout="horizontal"
                        dataSource={nodes}
                        renderItem={item => (
                            <List.Item

                                extra={
                                    <div style={{display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '8px'}}>
                                        {getActionDisplay(item)}
                                        {getLatencyDisplay(item.latency)}
                                    </div>
                                }
                                style={{
                                    borderRadius: '4px',
                                    marginBottom: '12px',
                                    transition: 'all 0.3s'
                                }}
                            >
                                <List.Item.Meta
                                    avatar={
                                        <Badge>
                                            <div style={{
                                                width: '40px',
                                                height: '40px',
                                                borderRadius: '50%',
                                                background: '#f0f7ff',
                                                display: 'flex',
                                                alignItems: 'center',
                                                justifyContent: 'center'
                                            }}>
                                                <ManOutlined style={{color: '#1890ff'}}/>
                                            </div>
                                        </Badge>
                                    }
                                    title={
                                        <div style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                                            <Text strong>{item.id}</Text>
                                        </div>
                                    }
                                    description={
                                        <div style={{marginTop: '8px'}}>
                                            <div style={{display: 'flex', alignItems: 'center', marginBottom: '4px'}}>
                                                <Text type="secondary">IP地址: </Text>
                                                <Text>{item.ip}</Text>
                                            </div>
                                        </div>
                                    }
                                />
                            </List.Item>
                        )}
                    />
                </Spin>
            </Content>

            <Footer style={{textAlign: 'center'}}>
                kvirnet ©{new Date().getFullYear()}
            </Footer>
        </Layout>
    );
};

export default App;

