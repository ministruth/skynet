import { ColSpanType } from '@ant-design/pro-card/es/typing';
import { Col, Row } from 'antd';

export interface SpanType {
  xs?: ColSpanType;
  md?: ColSpanType;
}

export interface RowItemProps {
  span?: SpanType;
  nextSpan?: SpanType;
  text: React.ReactNode;
  item: React.ReactNode;
  nospace?: boolean;
}
const RowItem: React.FC<RowItemProps> = (props) => {
  return (
    <Row style={props.nospace ? {} : { marginBottom: 24 }} align="middle">
      <Col xs={props.span?.xs} md={props.span?.md}>
        {props.text}
      </Col>
      <Col xs={props.nextSpan?.xs} md={props.nextSpan?.md}>
        {props.item}
      </Col>
    </Row>
  );
};

export default RowItem;
