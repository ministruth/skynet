import logo from '@/assets/logo.png';
import Footer from '@/components/footer';
import DeniedPage from '@/pages/403';
import { checkPerm, getIntl, UserPerm } from '@/utils';
import * as icons from '@ant-design/icons';
import ProLayout, { MenuDataItem } from '@ant-design/pro-layout';
import _ from 'lodash';
import React, { ReactNode, useEffect } from 'react';
import {
  Access,
  Helmet,
  history,
  Link,
  SelectLang,
  useAccess,
  useModel,
} from 'umi';

interface MainLayoutProps {
  title?: string;
  access?: string;
  perm?: UserPerm;
}

const loopMenuItem = (menus: MenuDataItem[]): MenuDataItem[] =>
  menus.map(({ name, icon, path, children }) => ({
    path: path,
    // @ts-ignore
    icon: icon && React.createElement(icons[icon]),
    name: name,
    routes: children && loopMenuItem(children),
  }));

const MainLayout: React.FC<MainLayoutProps> = (props) => {
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
        accessible={checkPerm(
          initialState?.signin,
          access,
          props.access,
          props.perm,
        )}
        fallback={DeniedPage()}
      >
        {props.children}
      </Access>
    );
  else children = props.children;

  return (
    <ProLayout
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
        request: async () => loopMenuItem(initialState?.menu as MenuDataItem[]),
      }}
      menuItemRender={(item, dom) => (
        <Link to={item.path ?? '/dashboard'}>{dom}</Link>
      )}
      rightContentRender={() => (
        <div data-lang>
          <SelectLang />
        </div>
      )}
      style={{
        height: '100vh',
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
