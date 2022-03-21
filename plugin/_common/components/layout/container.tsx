import { getIntl } from '@/utils';
import { PageContainer, PageContainerProps } from '@ant-design/pro-layout';
import { Route } from 'antd/lib/breadcrumb/Breadcrumb';

function render(
  route: Route,
  params: any,
  routes: Array<Route>,
  paths: Array<string>,
) {
  const last = routes.indexOf(route) === routes.length - 1;
  return last ? (
    <span>{route.breadcrumbName}</span>
  ) : (
    <a
      href={route.path === '' ? '/' : route.path}
      onClick={(e) => {
        e.preventDefault();
        history.pushState(null, '', route.path === '' ? '/' : route.path);
      }}
    >
      {route.breadcrumbName}
    </a>
  );
}

interface MainContainerProps {
  routes: Route[];
}

const MainContainer: React.FC<PageContainerProps & MainContainerProps> = (
  props,
) => {
  const intl = getIntl();
  props.routes.forEach((item) => {
    item.breadcrumbName = intl.get(item.breadcrumbName);
  });
  return (
    <PageContainer
      {...props}
      header={{
        breadcrumb: {
          itemRender: render,
          routes: [
            {
              path: '',
              breadcrumbName: 'Skynet',
            },
            ...props.routes,
          ],
        },
      }}
    >
      {props.children}
    </PageContainer>
  );
};

export default MainContainer;
