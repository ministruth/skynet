import DeniedPage from '@/pages/403';
import { checkPerm, getIntl, UserPerm } from '@/utils';
import { Access, Helmet, useModel } from '@umijs/max';
import React, { PropsWithChildren, ReactNode, useEffect } from 'react';

export interface MainLayoutProps {
  title?: string;
  access?: string;
  perm?: UserPerm;
}

const MainLayout: React.FC<PropsWithChildren<MainLayoutProps>> = (props) => {
  const { initialState, access } = useModel('@@qiankunStateFromMaster');
  const intl = getIntl();

  useEffect(() => {
    if (!initialState?.signin) window.location.href = '/';
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
    <>
      {props.title && (
        <Helmet>
          <title>{intl.get(props.title)}</title>
        </Helmet>
      )}
      {children}
    </>
  );
};

export default MainLayout;
