'use client';

import React from 'react';
import { Button, Divider, message } from 'antd';
import { WechatWorkOutlined, DingdingOutlined, QqOutlined } from '@ant-design/icons';

interface ThirdPartyLoginProps {
  className?: string;
}

export const ThirdPartyLogin: React.FC<ThirdPartyLoginProps> = ({ className }) => {
  const handleLogin = (provider: 'feishu' | 'wecom' | 'dingtalk') => {
    // 跳转到对应的第三方登录授权页面
    const redirectUri = encodeURIComponent(`${window.location.origin}/auth/callback/${provider}`);
    const authUrls: Record<string, string> = {
      feishu: `https://open.feishu.cn/open-apis/authen/v1/index?app_id=${process.env.NEXT_PUBLIC_FEISHU_APP_ID}&redirect_uri=${redirectUri}&state=${Date.now()}`,
      wecom: `https://open.work.weixin.qq.com/wwopen/sso/qrConnect?appid=${process.env.NEXT_PUBLIC_WECOM_APP_ID}&agentid=${process.env.NEXT_PUBLIC_WECOM_AGENT_ID}&redirect_uri=${redirectUri}&state=${Date.now()}`,
      dingtalk: `https://oapi.dingtalk.com/connect/oauth2/sns_authorize?appid=${process.env.NEXT_PUBLIC_DINGTALK_APP_ID}&response_type=code&scope=snsapi_login&redirect_uri=${redirectUri}&state=${Date.now()}`,
    };

    const authUrl = authUrls[provider];
    if (authUrl) {
      window.location.href = authUrl;
    } else {
      message.warning(`${provider === 'feishu' ? '飞书' : provider === 'wecom' ? '企业微信' : '钉钉'}登录暂未配置，请联系管理员`);
    }
  };

  return (
    <div className={className}>
      <Divider plain>其他登录方式</Divider>
      <div className="flex justify-center gap-4">
        <Button
          type="text"
          icon={<WechatWorkOutlined style={{ fontSize: 24, color: '#07C160' }} />}
          onClick={() => handleLogin('wecom')}
          size="large"
          title="企业微信登录"
        />
        <Button
          type="text"
          icon={<QqOutlined style={{ fontSize: 24, color: '#2663EB' }} />}
          onClick={() => handleLogin('feishu')}
          size="large"
          title="飞书登录"
        />
        <Button
          type="text"
          icon={<DingdingOutlined style={{ fontSize: 24, color: '#228AFF' }} />}
          onClick={() => handleLogin('dingtalk')}
          size="large"
          title="钉钉登录"
        />
      </div>
    </div>
  );
};
