import { getIntl } from '@/utils';
import { PageContainer, PageContainerProps } from '@ant-design/pro-layout';
import { BreadcrumbItemType } from 'antd/es/breadcrumb/Breadcrumb';

export interface MainContainerProps {
  routes: BreadcrumbItemType[];
}

const MainContainer: React.FC<PageContainerProps & MainContainerProps> = (
  props,
) => {
  const intl = getIntl();
  props.routes.forEach((item) => {
    if (typeof item.title === 'string') item.title = intl.get(item.title);
  });
  return (
    <PageContainer
      {...props}
      header={{
        breadcrumb: {
          items: [
            {
              title: 'Skynet',
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
