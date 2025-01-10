import logo from '@/assets/logo.png';
import Footer from '@/components/footer';
import DeniedPage from '@/pages/403';
import { checkPerm, getAPI, getIntl, postAPI, UserPerm } from '@/utils';
import * as icons from '@ant-design/icons';
import { LogoutOutlined } from '@ant-design/icons';
import ProLayout, { MenuDataItem } from '@ant-design/pro-layout';
import {
  Access,
  Helmet,
  history,
  Link,
  SelectLang,
  useAccess,
  useModel,
} from '@umijs/max';
import { Badge } from 'antd';
import _ from 'lodash';
import React, { PropsWithChildren, ReactNode, useEffect } from 'react';

export interface MainLayoutProps {
  title?: string;
  access?: string;
  perm?: UserPerm;
  postMenuData?: (item: MenuDataItem[]) => MenuDataItem[];
}

const loopMenuItem = (menus: MenuDataItem[]): MenuDataItem[] => {
  return menus.map(({ icon, children, badge, ...item }) => ({
    ...item,
    icon: icon && (
      <Badge size="small" count={badge}>
        {
          // @ts-ignore
          React.createElement(icons[icon])
        }
      </Badge>
    ),
    children: children && loopMenuItem(children),
  }));
};

const MainLayout: React.FC<PropsWithChildren<MainLayoutProps>> = (props) => {
  const { initialState } = useModel('@@initialState');
  const access = useAccess();
  const intl = getIntl();

  useEffect(() => {
    if (!initialState?.signin) history.push('/');
  }, []);

  let children: ReactNode;
  if (props.access && props.perm !== undefined)
    children = (
      <Access
        accessible={checkPerm(access, props.access, props.perm)}
        fallback={DeniedPage()}
      >
        {props.children}
      </Access>
    );
  else children = props.children;

  return (
    <ProLayout
      token={{
        header: {
          colorBgHeader: '#292f33',
          colorHeaderTitle: '#fff',
          colorTextMenu: '#dfdfdf',
          colorTextMenuSecondary: '#dfdfdf',
          colorTextMenuSelected: '#fff',
          colorBgMenuItemSelected: '#22272b',
          colorTextMenuActive: 'rgba(255,255,255,0.85)',
          colorTextRightActionsItem: '#dfdfdf',
        },
        colorTextAppListIconHover: '#fff',
        colorTextAppListIcon: '#dfdfdf',
        sider: {
          colorMenuBackground: '#fff',
          colorMenuItemDivider: '#dfdfdf',
          colorBgMenuItemHover: '#f6f6f6',
          colorTextMenu: '#595959',
          colorTextMenuSelected: '#242424',
          colorTextMenuActive: '#242424',
        },
      }}
      title="Skynet" // should enable to fix bug for small device
      pageTitleRender={() => {
        let title = '';
        if (props.title) title = intl.get(props.title);
        return title;
      }}
      logo={logo}
      onMenuHeaderClick={() => {
        history.push('/dashboard');
      }}
      menuProps={{
        selectedKeys: [_.trimEnd(window.location.pathname, '/')], // force use pathname only, fix auto select subpath
      }}
      fixSiderbar
      footerRender={() => <Footer />}
      menu={{
        locale: true,
        request: async (p) => {
          if (initialState?.signin)
            return getAPI('/menus').then((data) => {
              let d = data.data as MenuDataItem[];
              return loopMenuItem(
                props.postMenuData ? props.postMenuData(d) : d,
              );
            });
          else return [];
        },
      }}
      menuItemRender={(item, dom) => (
        <Link to={item.path ?? '/dashboard'}>{dom}</Link>
      )}
      actionsRender={() => [
        <SelectLang style={{ padding: 0 }} reload={true} />,
        <LogoutOutlined
          onClick={() => {
            postAPI('/signout', {}).then((rsp) => {
              if (rsp && rsp.code === 0)
                setTimeout(() => {
                  window.location.href = '/';
                }, 1000);
            });
          }}
        />,
      ]}
      // force cancel padding for microapp
      contentStyle={{ padding: 0 }}
      // make copyright at the bottom
      style={{
        minHeight: '100vh',
        height: 'max-content',
      }}
    >
      {/* fix bug when switch from microapp, title will disappear */}
      {props.title && (
        <Helmet>
          <title>{intl.get(props.title)}</title>
        </Helmet>
      )}
      {children}
    </ProLayout>
  );
};

export default MainLayout;
